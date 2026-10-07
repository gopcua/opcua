package harness

import (
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestSequenceCounterFollowsRetainAndAnswerWithSequenceNumber(t *testing.T) {
	env := New(t)
	sub := env.Subscription()
	if sequenceNumber := env.LastSequenceNumber(); sequenceNumber != 1 {
		t.Fatalf("the first answered Publish response carries sequence number %d, want 1: the counter must start at 1 when the subscription is created", sequenceNumber)
	}

	env.Server.WaitHeldPublish().Answer(sub, 7001)
	env.Server.WaitHeldPublish().Answer(sub, 7002)
	sub.Retain(10, 7100)
	sub.Retain(4, 7104)
	env.Server.WaitHeldPublish().Answer(sub, 7003)
	env.Server.WaitHeldPublish().AnswerWithSequenceNumber(sub, 20, 7200)
	env.Server.WaitHeldPublish().Answer(sub, 7201)
	env.Server.WaitHeldPublish().AnswerWithSequenceNumber(sub, 3, 7300)
	env.Server.WaitHeldPublish().Answer(sub, 7301)

	want := []answeredPublish{
		{sequenceNumber: 1, value: valueBeforeCut},
		{sequenceNumber: 2, value: 7001},
		{sequenceNumber: 3, value: 7002},
		{sequenceNumber: 11, value: 7003},
		{sequenceNumber: 20, value: 7200},
		{sequenceNumber: 21, value: 7201},
		{sequenceNumber: 3, value: 7300},
		{sequenceNumber: 22, value: 7301},
	}
	answered := waitAnsweredPublishes(env, want)
	got, wantSequences := answeredSequences(answered), answeredSequences(want)
	if !slices.Equal(got, wantSequences) {
		t.Fatalf("the answered Publish responses carry sequence numbers %v, want %v: Answer must use the counter and increment it, and Retain and AnswerWithSequenceNumber must raise it to max(counter, seq+1)", got, wantSequences)
	}
	for i := range answered {
		if answered[i].value != want[i].value {
			t.Fatalf("the Publish response with sequence number %d carries %d, want %d", answered[i].sequenceNumber, answered[i].value, want[i].value)
		}
	}

	for seq, wantValue := range map[uint32]int32{10: 7100, 4: 7104} {
		response, err := sendRepublish(env, sub.ID(), seq)
		if err != nil {
			t.Fatalf("the Republish for sequence number %d failed: %v", seq, err)
		}
		republish, isRepublish := response.(*ua.RepublishResponse)
		if !isRepublish {
			t.Fatalf("the Republish for sequence number %d answered with a %T, want a RepublishResponse", seq, response)
		}
		value, carries := dataChangeValue(republish.NotificationMessage)
		if !carries || value != wantValue {
			t.Fatalf("the Republish for sequence number %d answered with %v, want the retained value %d", seq, value, wantValue)
		}
	}
}

func waitAnsweredPublishes(env *Environment, want []answeredPublish) []answeredPublish {
	deadline := time.NewTimer(startTimeout)
	defer deadline.Stop()
	for {
		answered := env.Recorder.answeredPublishes()
		if len(answered) == len(want) {
			return answered
		}
		select {
		case <-deadline.C:
			env.t.Fatalf("the recorder saw %d answered Publish responses, want %d", len(answered), len(want))
			return nil
		case <-time.After(statePollInterval):
		}
	}
}

func answeredSequences(publishes []answeredPublish) []uint32 {
	sequenceNumbers := make([]uint32, 0, len(publishes))
	for _, publish := range publishes {
		sequenceNumbers = append(sequenceNumbers, publish.sequenceNumber)
	}
	return sequenceNumbers
}
