package message

import (
	"fmt"

	"github.com/blocklords/sds/common/data_type/key_value"
)

// ReplyStatus can be only as "OK" or "fail"
// It indicates whether the reply message is correct or not.
type ReplyStatus string

const (
	OK   ReplyStatus = "OK"
	FAIL ReplyStatus = "fail"
)

// SDS Service returns the reply. Anyone who sends a request to the SDS Service gets this message.
type Reply struct {
	Status     ReplyStatus        `json:"status"`     // message.OK or message.FAIL
	Message    string             `json:"message"`    // If Status is fail, then field will contain error message.
	Parameters key_value.KeyValue `json:"parameters"` // If Status is OK, then field will contain the parameters.
}

// Fail creates a new Reply as a failure
// It accepts the error message that explains the reason of the failure.
func Fail(message string) Reply {
	return Reply{Status: FAIL, Message: message, Parameters: key_value.Empty()}
}

// Validates the status of the reply.
// It should be ethier OK or fail.
func (reply *Reply) valid_status() error {
	if reply.Status != FAIL && reply.Status != OK {
		return fmt.Errorf("status is either '%s' or '%s', but given: '%s'", OK, FAIL, reply.Status)
	}

	return nil
}

// If the reply type is failure then
// THe message should be given too
func (reply *Reply) valid_fail() error {
	if reply.Status == FAIL && len(reply.Message) == 0 {
		return fmt.Errorf("failure should not have an empty message")
	}

	return nil
}

// IsOK returnes the Status of the message.
func (r *Reply) IsOK() bool { return r.Status == OK }

// ToString converts the Reply to the string format
func (reply *Reply) ToString() (string, error) {
	bytes, err := reply.ToBytes()
	if err != nil {
		return "", fmt.Errorf("reply.ToBytes: %w", err)
	}

	return string(bytes), nil
}

// ToBytes converts Reply to the sequence of bytes
func (reply *Reply) ToBytes() ([]byte, error) {
	err := reply.valid_fail()
	if err != nil {
		return nil, fmt.Errorf("failure validation: %w", err)
	}
	err = reply.valid_status()
	if err != nil {
		return nil, fmt.Errorf("status validation: %w", err)
	}

	kv, err := key_value.NewFromInterface(reply)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize Reply to key-value %v: %v", reply, err)
	}

	bytes, err := kv.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("serialized key-value.ToBytes: %w", err)
	}

	return bytes, nil
}

// ParseReply decodes the Zeromq messages into the Reply.
func ParseReply(msgs []string) (Reply, error) {
	msg := ToString(msgs)
	data, err := key_value.NewFromString(msg)
	if err != nil {
		return Reply{}, fmt.Errorf("key_value.NewFromString: %w", err)
	}

	reply, err := ParseJsonReply(data)
	if err != nil {
		return Reply{}, fmt.Errorf("ParseJsonReply: %w", err)
	}

	return reply, nil
}

// ParseJsonReply creates the 'Reply' message from a key value
func ParseJsonReply(dat key_value.KeyValue) (Reply, error) {
	var reply Reply
	err := dat.ToInterface(&reply)
	if err != nil {
		return Reply{}, fmt.Errorf("failed to serialize key-value to msg.Reply: %v", err)
	}

	// It will call
	// valid_fail(), valid_status() and
	// check for missing values
	_, err = reply.ToBytes()
	if err != nil {
		return Reply{}, fmt.Errorf("validation: %w", err)
	}

	return reply, nil
}
