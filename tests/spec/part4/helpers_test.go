package part4

import (
	"time"

	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/gomega"
)

func requestsOfType[T ua.Request](records []spectest.ServiceRecord[ua.Request]) []spectest.ServiceRecord[ua.Request] {
	var matched []spectest.ServiceRecord[ua.Request]
	for _, record := range records {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if _, is := message.(T); is {
			matched = append(matched, record)
		}
	}
	return matched
}

func answerTo(request spectest.ServiceRecord[ua.Request], responses []spectest.ServiceRecord[ua.Response]) (spectest.ServiceRecord[ua.Response], bool) {
	for _, response := range responses {
		if response.Connection == request.Connection && response.RequestID == request.RequestID {
			return response, true
		}
	}
	return spectest.ServiceRecord[ua.Response]{}, false
}

func statusOf(response spectest.ServiceRecord[ua.Response]) (ua.StatusCode, bool) {
	message, decoded := response.Message()
	if !decoded {
		return 0, false
	}
	return message.Header().ServiceResult, true
}

func republishForSequence(records []spectest.ServiceRecord[ua.Request], sequenceNumber uint32) (spectest.ServiceRecord[ua.Request], bool) {
	for _, record := range requestsOfType[*ua.RepublishRequest](records) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if republish, is := message.(*ua.RepublishRequest); is && republish.RetransmitSequenceNumber == sequenceNumber {
			return record, true
		}
	}
	return spectest.ServiceRecord[ua.Request]{}, false
}

func preCutSessionToken(env *spectest.Environment) *ua.NodeID {
	for _, record := range requestsOfType[*ua.ActivateSessionRequest](env.Recorder.Requests()) {
		message, decoded := record.Message()
		if decoded {
			return message.Header().AuthenticationToken
		}
	}
	return nil
}

func notificationCarryingValue(env *spectest.Environment, value int32) (spectest.Notification, bool) {
	for _, notification := range env.Recorder.Notifications() {
		if notification.Value == value {
			return notification, true
		}
	}
	return spectest.Notification{}, false
}

func badMessageNotAvailableAnswer(env *spectest.Environment, m spectest.Mark) (spectest.ServiceRecord[ua.Response], bool) {
	responses := env.Recorder.ResponsesSince(m)
	for _, republish := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if answer, answered := answerTo(republish, responses); answered {
			if status, decoded := statusOf(answer); decoded && status == ua.StatusBadMessageNotAvailable {
				return answer, true
			}
		}
	}
	return spectest.ServiceRecord[ua.Response]{}, false
}

func waitAnsweredBadMessageNotAvailable(env *spectest.Environment, m spectest.Mark) spectest.ServiceRecord[ua.Response] {
	var answer spectest.ServiceRecord[ua.Response]
	Eventually(func(g Gomega) {
		var answered bool
		answer, answered = badMessageNotAvailableAnswer(env, m)
		g.Expect(answered).To(BeTrue(), "client sent no Republish request that the server answered Bad_MessageNotAvailable")
	}, 15*time.Second).Should(Succeed())
	return answer
}
