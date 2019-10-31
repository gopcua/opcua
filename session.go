package opcua

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// Session is a OPC/UA session as described in Part 4, 5.6.
type Session struct {
	c   *Client
	cfg *uasc.SessionConfig

	// resp is the response to the CreateSession request which contains all
	// necessary parameters to activate the session.
	resp *ua.CreateSessionResponse

	// serverCertificate is the certificate used to generate the signatures for
	// the ActivateSessionRequest methods
	serverCertificate []byte

	// serverNonce is the secret nonce received from the server during Create and Activate
	// Session response. Used to generate the signatures for the ActivateSessionRequest
	// and User Authorization
	serverNonce []byte

	publishEngine *publishEngine

	keepAliveManager *KeepAliveManager
	cancelKeepAlive  context.CancelFunc
}

func (s *Session) isChannelValid() bool {
	if s.c == nil {
		debug.Printf("opcua: warning SessionClient is null ?")
		return false
	}
	return s.c.sechan != nil && s.c.sechan.IsOpen()
}

func (s *Session) repairSession() error {

	s.publishEngine.Suspend()
	s.keepAliveManager.Suspend()
	if _, err := s.c.DetachSession(); err != nil {
		return err
	}

	debug.Printf("opcua: trying to reactivate existing session")

	err := s.c.ActivateSession(s)
	if err != nil {
		debug.Printf("opcua: session reactivation failed, recreating new session")
		if err = s.repairSessionByRecreatingNewSession(); err != nil {
			return err
		}

		if err := s.c.ActivateSession(s); err != nil {
			_ = s.c.Close()
			return err
		}

		debug.Printf("opcua: trying to transfert existing subscriptions")
		subscriptionIDs := s.publishEngine.SubscriptionIDs()
		debug.Printf("opcua: session subscriptionCount = %d", len(subscriptionIDs))

		res, err := s.transferSubscriptions(subscriptionIDs)
		subscriptionsToRecreate := []uint32{}
		subscriptionsToRepair := []uint32{}

		if err != nil {
			debug.Printf("opcua: transfert subscriptions has failed, %s", err.Error())
			subscriptionsToRecreate = subscriptionIDs
		} else {
			for id := range res {
				transferResult := res[id]
				if transferResult.StatusCode == ua.StatusBadSubscriptionIDInvalid {
					debug.Printf("opcua: warning suscription (id: %d), should be recreated", id)
					subscriptionsToRecreate = append(subscriptionsToRecreate, subscriptionIDs[id])
				} else {
					debug.Printf(
						"opcua: subscription (id: %d) can be repaired and available",
						transferResult.AvailableSequenceNumbers[id],
					)
					subscriptionsToRepair = append(subscriptionsToRepair, subscriptionIDs[id])
				}
			}
		}
		if len(subscriptionsToRecreate) > 0 {
			if err := s.repairSubscriptionsByRecreatingNewSubscriptions(subscriptionIDs); err != nil {
				return err
			}
		}
		if len(subscriptionsToRepair) > 0 {
			if err := s.repairSubscriptions(subscriptionsToRepair...); err != nil {
				return err
			}
		}

		debug.Printf("opcua: transfer subscriptions done")
	} else {
		debug.Printf("opcua: existing session reactivated trying to repair subscriptions")
		if err := s.repairSubscriptions(); err != nil {
			return err
		}
		debug.Printf("opcua: subscriptions repaired")
	}

	// Force resume publish engine
	s.publishEngine.Resume()
	s.keepAliveManager.Resume()

	return nil
}

func (s *Session) repairSessionByRecreatingNewSession() error {
	if s.c.sechan == nil {
		return errors.Errorf("secure channel not connected")
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	cfg := s.c.sessionCfg

	req := &ua.CreateSessionRequest{
		ClientDescription:       cfg.ClientDescription,
		EndpointURL:             s.c.endpointURL,
		SessionName:             fmt.Sprintf("gopcua-%d", time.Now().UnixNano()),
		ClientNonce:             nonce,
		ClientCertificate:       s.c.cfg.Certificate,
		RequestedSessionTimeout: float64(cfg.SessionTimeout / time.Millisecond),
	}

	// for the CreateSessionRequest the authToken is always nil.
	// use c.sechan.Send() to enforce this.
	err := s.c.sechan.SendRequest(req, nil, func(v interface{}) error {
		var res *ua.CreateSessionResponse
		if err := safeAssign(v, &res); err != nil {
			return err
		}

		err := s.c.sechan.VerifySessionSignature(res.ServerCertificate, nonce, res.ServerSignature.Signature)
		if err != nil {
			log.Printf("error verifying session signature: %s", err)
			return nil
		}

		// Ensure we have a valid identity token that the server will accept before trying to activate a session
		if s.c.sessionCfg.UserIdentityToken == nil {
			opt := AuthAnonymous()
			opt(s.c.cfg, s.c.sessionCfg)

			p := anonymousPolicyID(res.ServerEndpoints)
			opt = AuthPolicyID(p)
			opt(s.c.cfg, s.c.sessionCfg)
		}

		s.resp = res
		s.serverNonce = res.ServerNonce
		s.serverCertificate = res.ServerCertificate

		return nil
	})
	return err
}

func (s *Session) repairSubscriptions(subscriptionIDs ...uint32) error {
	return s.publishEngine.RepairSubscriptions(subscriptionIDs...)
}

func (s *Session) repairSubscriptionsByRecreatingNewSubscriptions(subscriptionIDs []uint32) error {

	for _, subscriptionID := range subscriptionIDs {
		if !s.publishEngine.HasSubscription(subscriptionID) {
			debug.Printf("opcua: cannot recreate subscription")
			continue
		}
		subscription := s.publishEngine.GetSubscription(subscriptionID)

		debug.Printf("opcua: recreating subscription (id = %d)", subscriptionID)
		if err := subscription.recreateSubscriptionAndMonitoredItem(); err != nil {
			debug.Printf("opcua: recreate subscription failed")
			return err
		}
		debug.Printf("opcua: recreating subscription and monitored item done")
	}

	return nil
}

func (s *Session) transferSubscriptions(subscriptionIDs []uint32) ([]*ua.TransferResult, error) {

	req := &ua.TransferSubscriptionsRequest{
		SubscriptionIDs:   subscriptionIDs,
		SendInitialValues: false,
	}
	var res *ua.TransferSubscriptionsResponse
	err := s.c.sechan.SendRequest(req, s.resp.AuthenticationToken, func(v interface{}) error {
		if err := safeAssign(v, &res); err != nil {
			return err
		}

		if err := s.c.CloseSession(); err != nil {
			// try to close the newly created session but report
			// only the initial error.
			_ = s.c.closeSession(s)
			return err
		}
		s.c.session.Store(s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res.Results, nil
}

func (s *Session) publish(acks []*ua.SubscriptionAcknowledgement, timeout time.Duration) (*ua.PublishResponse, error) {
	if acks == nil {
		acks = []*ua.SubscriptionAcknowledgement{}
	}
	req := &ua.PublishRequest{
		SubscriptionAcknowledgements: acks,
	}
	var res *ua.PublishResponse
	err := s.c.sendWithTimeout(req, s.publishTimeout(timeout), func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

func (s *Session) publishTimeout(timeout time.Duration) time.Duration {
	if timeout > uasc.MaxTimeout {
		return uasc.MaxTimeout
	}
	if timeout < s.c.cfg.RequestTimeout {
		return s.c.cfg.RequestTimeout
	}
	return timeout
}

func (s *Session) republish(req *ua.RepublishRequest) (*ua.RepublishResponse, error) {
	var res *ua.RepublishResponse
	err := s.c.sechan.SendRequest(req, s.resp.AuthenticationToken, func(v interface{}) error {
		if err := safeAssign(v, &res); err != nil {
			return err
		}
		if res.ResponseHeader.ServiceResult != ua.StatusOK {
			return errors.Errorf(res.ResponseHeader.ServiceResult.Error())
		}
		return nil
	})
	return res, err
}
