package rules

import (
	"slices"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/ua"

	"github.com/onsi/gomega"
)

// Rule is one Part 4 behaviour the specs assert: a stable Name the
// known-defect table cites, the Clause label and Keyword of its
// sentence in the standard, and the Check that asserts it against the
// context the caller observed.
type Rule struct {
	Name    string
	Clause  string
	Keyword string
	Check   func(c Context)
}

// Context is what a rule's Check needs: the environment the scenario
// ran in, the Mark taken before the fault, the last sequence number
// the client delivered, the subscription the client held before the
// fault and the one it recreated when it did, the server the client
// should end on, how a transfer answer refuses when the scenario
// scripted one, and the value the scenario answered on the new
// subscription.
type Context struct {
	Env             *harness.Environment
	Mark            harness.Mark
	LastSeq         uint32
	Sub             harness.Subscription
	Recreated       harness.Subscription
	Server          *harness.ScriptedServer
	TransferRefusal func(harness.ServiceRecord[ua.Response]) bool
	Value           int32
	// CyclesCompleted and CyclesWanted say how many cancel-then-subscribe
	// cycles the workload drove and how many it wanted to complete; a
	// client that parks its publish loop stops them early.
	CyclesCompleted int
	CyclesWanted    int
	// HeldOrder is the recorded Order of the Publish request the workload
	// held past the client's publish timeout; HeldAnswerOrder is the
	// recorded Order of the response it then sent to it, or 0 when it
	// never answered it.
	HeldOrder       int
	HeldAnswerOrder int
}

// ReactivatesSession: after a transport loss the client re-activates
// the session it had, carrying the authentication token the server
// issued before the loss.
var ReactivatesSession = Rule{
	Name:    "ReactivatesSession",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		reactivationAnswer(c.Env, c.Mark)
	},
}

// CreatesNoSession: the client creates no new session before the
// re-activation of the old one was answered, and none after it
// succeeded.
var CreatesNoSession = Rule{
	Name:    "CreatesNoSession",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		reactivation := reactivationAnswer(c.Env, c.Mark)
		gomega.Expect(slices.ContainsFunc(RequestsOfType[*ua.CreateSessionRequest](c.Env.Recorder.RequestsSince(c.Mark)), func(request harness.ServiceRecord[ua.Request]) bool {
			return request.Order < reactivation.Order
		})).To(gomega.BeFalse(), "client sent a CreateSession request before the server answered ActivateSession")
		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(RequestsOfType[*ua.CreateSessionRequest](c.Env.Recorder.RequestsSince(c.Mark))).To(gomega.BeEmpty(), "client sent a CreateSession request after the server answered ActivateSession")
		}, 2*time.Second).Should(gomega.Succeed())
	},
}

// RepublishesFromNextSequence: the client republishes from the next
// expected sequence number, incrementing, until the server answers
// Bad_MessageNotAvailable.
var RepublishesFromNextSequence = Rule{
	Name:    "RepublishesFromNextSequence",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		gomega.Eventually(func(g gomega.Gomega) {
			requests := c.Env.Recorder.RequestsSince(c.Mark)
			responses := c.Env.Recorder.ResponsesSince(c.Mark)
			complete := false
			first, firstSent := RepublishForSequence(requests, c.LastSeq+1)
			if firstSent {
				second, secondSent := RepublishForSequence(requests, c.LastSeq+2)
				if secondSent && second.Order > first.Order {
					if answer, answered := AnswerTo(second, responses); answered {
						if status, decoded := StatusOf(answer); decoded && status == ua.StatusBadMessageNotAvailable {
							complete = true
						}
					}
				}
			}
			g.Expect(complete).To(gomega.BeTrue(), "no Republish request for sequence number %d followed by one for %d answered Bad_MessageNotAvailable was recorded after the reconnect", c.LastSeq+1, c.LastSeq+2)
		}, 15*time.Second).Should(gomega.Succeed())
		gomega.Consistently(func(g gomega.Gomega) {
			republishes := RequestsOfType[*ua.RepublishRequest](c.Env.Recorder.RequestsSince(c.Mark))
			g.Expect(republishes).To(gomega.HaveLen(2), "client sent more than the two expected Republish requests after the cut: %d recorded", len(republishes))
		}, 2*time.Second).Should(gomega.Succeed())
	},
}

// SendsNoPublishBeforeNotAvailable: the client sends no Publish
// request on its new connection until the Republish the server
// answered Bad_MessageNotAvailable.
var SendsNoPublishBeforeNotAvailable = Rule{
	Name:    "SendsNoPublishBeforeNotAvailable",
	Clause:  "P4-6.7",
	Keyword: "should",
	Check: func(c Context) {
		notAvailableAnswer := WaitAnsweredBadMessageNotAvailable(c.Env, c.Mark)
		gomega.Expect(slices.ContainsFunc(RequestsOfType[*ua.PublishRequest](c.Env.Recorder.RequestsSince(c.Mark)), func(request harness.ServiceRecord[ua.Request]) bool {
			return request.Connection == notAvailableAnswer.Connection && request.Order < notAvailableAnswer.Order
		})).To(gomega.BeFalse(), "client sent a Publish request on the new connection before the Republish was answered Bad_MessageNotAvailable")
	},
}

// SendsNoTransferForOwnSubscription: the client sends no
// TransferSubscriptions request for a subscription its own session
// owns.
var SendsNoTransferForOwnSubscription = Rule{
	Name:    "SendsNoTransferForOwnSubscription",
	Clause:  "P4-5.14.7.4",
	Keyword: "shall",
	Check: func(c Context) {
		notAvailableAnswer := WaitAnsweredBadMessageNotAvailable(c.Env, c.Mark)
		gomega.Expect(slices.ContainsFunc(RequestsOfType[*ua.TransferSubscriptionsRequest](c.Env.Recorder.RequestsSince(c.Mark)), func(request harness.ServiceRecord[ua.Request]) bool {
			return request.Order < notAvailableAnswer.Order
		})).To(gomega.BeFalse(), "client sent a TransferSubscriptions request before the Republish was answered Bad_MessageNotAvailable")
		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(RequestsOfType[*ua.TransferSubscriptionsRequest](c.Env.Recorder.RequestsSince(c.Mark))).To(gomega.BeEmpty(), "client sent a TransferSubscriptions request for a subscription its own session owns")
		}, 2*time.Second).Should(gomega.Succeed())
	},
}

// KeepsSubscriptionID: the client republishes under the subscription
// id it had before the cut, and creates no new subscription instead.
var KeepsSubscriptionID = Rule{
	Name:    "KeepsSubscriptionID",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		WaitAnsweredBadMessageNotAvailable(c.Env, c.Mark)
		wrongID := false
		for _, republish := range RequestsOfType[*ua.RepublishRequest](c.Env.Recorder.RequestsSince(c.Mark)) {
			message, decoded := republish.Message()
			if !decoded {
				continue
			}
			if request, is := message.(*ua.RepublishRequest); is && request.SubscriptionID != c.Sub.ID() {
				wrongID = true
			}
		}
		gomega.Expect(wrongID).To(gomega.BeFalse(), "client sent a Republish request naming a subscription id other than %d", c.Sub.ID())
		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(RequestsOfType[*ua.CreateSubscriptionRequest](c.Env.Recorder.RequestsSince(c.Mark))).To(gomega.BeEmpty(), "client created a new subscription instead of keeping subscription %d", c.Sub.ID())
		}, 2*time.Second).Should(gomega.Succeed())
	},
}

// CreatesSessionAfterActivateTimedOut: after the ActivateSession the
// client sent was not answered within the request timeout, the client
// creates a new session. A timeout is a failure, and Part 4 lets the
// client create a new session once ActivateSession has failed.
var CreatesSessionAfterActivateTimedOut = Rule{
	Name:    "CreatesSessionAfterActivateTimedOut",
	Clause:  "P4-6.7",
	Keyword: "should",
	Check: func(c Context) {
		sessionToken := PreCutSessionToken(c.Env)
		gomega.Expect(sessionToken).NotTo(gomega.BeNil(), "the recorder saw no ActivateSession request before the cut")
		gomega.Eventually(func(g gomega.Gomega) {
			requests := c.Env.Recorder.RequestsSince(c.Mark)
			complete := false
			for _, record := range RequestsOfType[*ua.ActivateSessionRequest](requests) {
				message, decoded := record.Message()
				if !decoded || !message.Header().AuthenticationToken.Equal(sessionToken) {
					continue
				}
				for _, create := range RequestsOfType[*ua.CreateSessionRequest](requests) {
					if create.Order > record.Order {
						complete = true
					}
				}
			}
			g.Expect(complete).To(gomega.BeTrue(),
				"client sent no CreateSession after the ActivateSession that timed out; requests since the mark: %v", RequestTypeNames(requests))
		}, 15*time.Second).Should(gomega.Succeed())
	},
}

// CreatesSessionOnlyAfterActivateFailed: the client creates a new
// session only after trying to activate the old one and being refused
// Bad_SessionIdInvalid.
var CreatesSessionOnlyAfterActivateFailed = Rule{
	Name:    "CreatesSessionOnlyAfterActivateFailed",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		gomega.Eventually(func(g gomega.Gomega) {
			requests := c.Env.Recorder.RequestsSince(c.Mark)
			responses := c.Env.Recorder.ResponsesSince(c.Mark)
			createSessions := RequestsOfType[*ua.CreateSessionRequest](requests)
			preceded := false
			if len(createSessions) > 0 {
				for _, record := range RequestsOfType[*ua.ActivateSessionRequest](requests) {
					answer, answered := AnswerTo(record, responses)
					if !answered {
						continue
					}
					if status, decoded := StatusOf(answer); decoded && status == ua.StatusBadSessionIDInvalid && answer.Order < createSessions[0].Order {
						preceded = true
					}
				}
			}
			g.Expect(preceded).To(gomega.BeTrue(), "client created a new session without first trying to activate the old one and being refused Bad_SessionIdInvalid; requests after the first cut: %v", RequestTypeNames(requests))
		}, 15*time.Second).Should(gomega.Succeed())
	},
}

// RecreatesAfterRefusal: after the server refuses the subscription —
// a TransferSubscriptions answered the way TransferRefusal
// recognizes, or, when TransferRefusal is nil, a Republish answered
// Bad_SubscriptionIdInvalid — the client creates a new subscription
// and re-monitors its node on it.
var RecreatesAfterRefusal = Rule{
	Name:    "RecreatesAfterRefusal",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		if c.TransferRefusal != nil {
			recreateAfterTransferRefusal(c)
			return
		}
		recreateAfterRepublishRefusal(c)
	},
}

func recreateAfterTransferRefusal(c Context) {
	var createRequest harness.ServiceRecord[ua.Request]
	var createdID uint32
	gomega.Eventually(func(g gomega.Gomega) {
		transferAnswer, create, createAnswer, complete := answeredTransferThenNewSubscription(c.Env, c.Mark, c.Sub.ID(), c.TransferRefusal)
		g.Expect(complete).To(gomega.BeTrue(),
			"no TransferSubscriptions request for subscription %d refused per the scripted answer followed by an answered CreateSubscription request was recorded", c.Sub.ID())
		createRequest = create
		answerMessage, answerDecoded := createAnswer.Message()
		if response, isCreate := answerMessage.(*ua.CreateSubscriptionResponse); answerDecoded && isCreate {
			createdID = response.SubscriptionID
		}
		_, _ = transferAnswer, answerMessage
	}, 15*time.Second).Should(gomega.Succeed())
	monitoredItemRecreated(c, createdID, createRequest.Order)
}

func recreateAfterRepublishRefusal(c Context) {
	var republishAnswer harness.ServiceRecord[ua.Response]
	gomega.Eventually(func(g gomega.Gomega) {
		requests := c.Env.Recorder.RequestsSince(c.Mark)
		responses := c.Env.Recorder.ResponsesSince(c.Mark)
		complete := false
		if republish, sent := RepublishForSequence(requests, c.LastSeq+1); sent {
			if answer, answered := AnswerTo(republish, responses); answered {
				if status, decoded := StatusOf(answer); decoded && status == ua.StatusBadSubscriptionIDInvalid {
					republishAnswer = answer
					complete = true
				}
			}
		}
		g.Expect(complete).To(gomega.BeTrue(), "no Republish request for sequence number %d answered Bad_SubscriptionIdInvalid was recorded after the reconnect", c.LastSeq+1)
	}, 15*time.Second).Should(gomega.Succeed())
	node := MonitoredNode(c.Env)
	gomega.Expect(node).NotTo(gomega.BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
	var createSubscription harness.ServiceRecord[ua.Request]
	var createdID uint32
	gomega.Eventually(func(g gomega.Gomega) {
		requests := c.Env.Recorder.RequestsSince(c.Mark)
		responses := c.Env.Recorder.ResponsesSince(c.Mark)
		sent := false
		for _, request := range RequestsOfType[*ua.CreateSubscriptionRequest](requests) {
			if request.Order > republishAnswer.Order {
				if answer, answered := AnswerTo(request, responses); answered {
					answerMessage, answerDecoded := answer.Message()
					if response, isCreate := answerMessage.(*ua.CreateSubscriptionResponse); answerDecoded && isCreate {
						createSubscription = request
						createdID = response.SubscriptionID
						sent = true
					}
				}
				break
			}
		}
		g.Expect(sent).To(gomega.BeTrue(), "client sent no CreateSubscription request answered with a subscription id after the Republish was answered Bad_SubscriptionIdInvalid")
	}, 15*time.Second).Should(gomega.Succeed())
	monitoredItemRecreated(c, createdID, createSubscription.Order)
}

func monitoredItemRecreated(c Context, id uint32, orderFloor int) {
	node := MonitoredNode(c.Env)
	gomega.Expect(node).NotTo(gomega.BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
	gomega.Eventually(func(g gomega.Gomega) {
		requests := c.Env.Recorder.RequestsSince(c.Mark)
		sent := false
		for _, record := range RequestsOfType[*ua.CreateMonitoredItemsRequest](requests) {
			if record.Order <= orderFloor {
				continue
			}
			message, decoded := record.Message()
			if !decoded {
				continue
			}
			request, is := message.(*ua.CreateMonitoredItemsRequest)
			if !is || request.SubscriptionID != id || len(request.ItemsToCreate) == 0 {
				continue
			}
			item := request.ItemsToCreate[0]
			if item == nil || item.ItemToMonitor == nil || !item.ItemToMonitor.NodeID.Equal(node) {
				continue
			}
			sent = true
			break
		}
		g.Expect(sent).To(gomega.BeTrue(), "client sent no CreateMonitoredItems request for the monitored node on subscription %d", id)
	}, 15*time.Second).Should(gomega.Succeed())
}

// RepublishesRecreatedFromOne: the client republishes the recreated
// subscription starting from sequence number one.
var RepublishesRecreatedFromOne = Rule{
	Name:    "RepublishesRecreatedFromOne",
	Clause:  "P4-6.7",
	Keyword: "shall",
	Check: func(c Context) {
		gomega.Eventually(func(g gomega.Gomega) {
			requests := c.Env.Recorder.RequestsSince(c.Mark)
			for _, record := range RequestsOfType[*ua.RepublishRequest](requests) {
				message, decoded := record.Message()
				if !decoded {
					continue
				}
				if republish, is := message.(*ua.RepublishRequest); is && republish.SubscriptionID == c.Recreated.ID() {
					g.Expect(republish.RetransmitSequenceNumber).To(gomega.Equal(uint32(1)),
						"the first Republish for the recreated subscription asks for sequence number %d, want 1", republish.RetransmitSequenceNumber)
					return
				}
			}
			g.Expect(true).To(gomega.BeFalse(), "no Republish for the recreated subscription was recorded yet; requests since the mark taken before the first cut: %v", RequestTypeNames(requests))
		}, 15*time.Second).Should(gomega.Succeed())
	},
}

// RepublishesWithinTimeoutAfterPublishTimeout: when the server holds
// a Publish until the client's publishing times out, the client sends
// the next Publish within that timeout instead of waiting forever.
// The check reads recorded traffic: a Publish request on the held
// one's connection after its order and before the answer the workload
// gave it, or after it when the workload gave none.
var RepublishesWithinTimeoutAfterPublishTimeout = Rule{
	Name:    "RepublishesWithinTimeoutAfterPublishTimeout",
	Clause:  "P4-5.14.1.2",
	Keyword: "should",
	Check: func(c Context) {
		sent := false
		for _, record := range RequestsOfType[*ua.PublishRequest](c.Env.Recorder.RequestsSince(c.Mark)) {
			if record.Connection != c.Env.Recorder.ConnectionOfOrder(c.HeldOrder) {
				continue
			}
			if record.Order > c.HeldOrder && (c.HeldAnswerOrder == 0 || record.Order < c.HeldAnswerOrder) {
				sent = true
				break
			}
		}
		gomega.Expect(sent).To(gomega.BeTrue(), "the client sent no Publish request on connection %d after the held one (order %d) and before its answer (order %d), so it never timed out the held Publish",
			c.Env.Recorder.ConnectionOfOrder(c.HeldOrder), c.HeldOrder, c.HeldAnswerOrder)
	},
}

// KeepsPublishingAfterCancelThenSubscribe: after the client cancels
// its only subscription and creates a new one while a Publish is
// held, the client answers arriving values: it completes every
// cancel-then-subscribe cycle the workload drives, keeps sending
// Publish requests and delivers the value answered on the new
// subscription.
var KeepsPublishingAfterCancelThenSubscribe = Rule{
	Name:    "KeepsPublishingAfterCancelThenSubscribe",
	Clause:  "P4-5.14.1.2",
	Keyword: "should",
	Check: func(c Context) {
		if c.CyclesWanted > 0 {
			gomega.Expect(c.CyclesCompleted).To(gomega.Equal(c.CyclesWanted),
				"the client parked its publish loop after cycle %d of %d: the cancel and the resume raced",
				c.CyclesCompleted+1, c.CyclesWanted)
		}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(RequestsOfType[*ua.PublishRequest](c.Env.Recorder.RequestsSince(c.Mark))).NotTo(gomega.BeEmpty(),
				"the client sent no further Publish request within %s after the held one was answered, so it did not keep publishing", 5*time.Second)
		}, 5*time.Second).Should(gomega.Succeed())
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(c.Env.ReceivedSince(c.Mark)).To(gomega.ContainElement(c.Value),
				"the client delivered no value answered on the new subscription; delivered since the mark: %v", c.Env.ReceivedSince(c.Mark))
		}, 5*time.Second).Should(gomega.Succeed())
	},
}

// RepublishesSkippedSequence: for every sequence number the server
// skipped on a subscription — a number missing between two consecutive
// notifications the client received on it — the client sends a Republish
// for the missing number. No gap means the rule holds vacuously.
var RepublishesSkippedSequence = Rule{
	Name:    "RepublishesSkippedSequence",
	Clause:  "P4-6.7",
	Keyword: "should",
	Check: func(c Context) {
		requests := c.Env.Recorder.RequestsSince(c.Mark)
		responses := c.Env.Recorder.ResponsesSince(c.Mark)
		for _, missing := range skippedSequenceNumbers(receivedNotifications(requests, responses)) {
			_, sent := RepublishForSequence(requests, missing)
			gomega.Expect(sent).To(gomega.BeTrue(),
				"client sent no Republish request for sequence number %d that the server skipped", missing)
		}
	},
}

// PublishesAgainAfterTooManyPublishRequests: for every Publish the
// server answered Bad_TooManyPublishRequests, the client sends another
// Publish on the same session, and a value answered after it is
// delivered. No such answer means the rule holds vacuously.
var PublishesAgainAfterTooManyPublishRequests = Rule{
	Name:    "PublishesAgainAfterTooManyPublishRequests",
	Clause:  "P4-5.14.5",
	Keyword: "should",
	Check: func(c Context) {
		requests := c.Env.Recorder.RequestsSince(c.Mark)
		responses := c.Env.Recorder.ResponsesSince(c.Mark)
		for _, refused := range publishesAnsweredTooMany(requests, responses) {
			refusedMessage, decoded := refused.request.Message()
			if !decoded {
				continue
			}
			token := refusedMessage.Header().AuthenticationToken
			var later harness.ServiceRecord[ua.Request]
			found := false
			for _, publish := range RequestsOfType[*ua.PublishRequest](requests) {
				if publish.Order <= refused.answer.Order {
					continue
				}
				publishMessage, publishDecoded := publish.Message()
				if !publishDecoded || !publishMessage.Header().AuthenticationToken.Equal(token) {
					continue
				}
				later = publish
				found = true
				break
			}
			gomega.Expect(found).To(gomega.BeTrue(),
				"client sent no Publish after the server answered its Publish (connection %d, request id %d) with Bad_TooManyPublishRequests",
				refused.request.Connection, refused.request.RequestID)
			delivered := false
			for _, response := range responses {
				if response.Fate != harness.Forwarded || response.Order <= later.Order {
					continue
				}
				responseMessage, responseDecoded := response.Message()
				if !responseDecoded {
					continue
				}
				var notification *ua.NotificationMessage
				switch answer := responseMessage.(type) {
				case *ua.PublishResponse:
					notification = answer.NotificationMessage
				case *ua.RepublishResponse:
					notification = answer.NotificationMessage
				default:
					continue
				}
				value, carries := notificationValue(notification)
				if carries && slices.Contains(c.Env.Received(), value) {
					delivered = true
					break
				}
			}
			gomega.Expect(delivered).To(gomega.BeTrue(),
				"no value answered after the Publish that followed the server answering its Publish (connection %d, request id %d) with Bad_TooManyPublishRequests was delivered",
				refused.request.Connection, refused.request.RequestID)
		}
	},
}

// All lists every rule the package holds, each exactly once.
func All() []Rule {
	return []Rule{
		ReactivatesSession,
		CreatesNoSession,
		RepublishesFromNextSequence,
		SendsNoPublishBeforeNotAvailable,
		SendsNoTransferForOwnSubscription,
		KeepsSubscriptionID,
		CreatesSessionAfterActivateTimedOut,
		CreatesSessionOnlyAfterActivateFailed,
		RecreatesAfterRefusal,
		RepublishesRecreatedFromOne,
		RepublishesWithinTimeoutAfterPublishTimeout,
		KeepsPublishingAfterCancelThenSubscribe,
		RepublishesSkippedSequence,
		PublishesAgainAfterTooManyPublishRequests,
	}
}
