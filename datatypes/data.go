package datatypes

// Data is an interface to handle any kind of OPC UA data types.
type Data interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	SerializeTo([]byte) error
	Len() int
	DataType() uint16
}
