package tunnel

import (
	"bytes"
	"encoding/binary"
	"io"
)

type QosFlowDescs []QosFlowDesc

func (q *QosFlowDescs) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	for _, desc := range *q {
		if descBuf, err := desc.MarshalBinary(); err != nil {
			return nil, err
		} else {
			_, err = buf.Write(descBuf)
			if err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

func (q *QosFlowDescs) UnmarshalBinary(b []byte) error {
	*q = make(QosFlowDescs, 0)
	buf := bytes.NewBuffer(b)
	for {
		if desc, err := parseQosFlowDesc(buf); err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		} else {
			*q = append(*q, *desc)
		}
	}
}

type QosFlowOperationCode uint8

const (
	OperationCodeCreateNewQosFlowDescription QosFlowOperationCode = iota + 1
	OperationCodeDeleteExistingQosFlowDescription
	OperationCodeModifyExistingQosFlowDescription
)

// QosFlowDesc represnets the Qos Flow Description in NAS
type QosFlowDesc struct {
	QFI           uint8
	OperationCode QosFlowOperationCode
	Parameters    QosFlowParameterList
}

func (q *QosFlowDesc) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// QFI
	if err := binary.Write(buf, binary.BigEndian, q.QFI); err != nil {
		return nil, err
	}

	// Operation Code
	if err := binary.Write(buf, binary.BigEndian, q.OperationCode<<5); err != nil {
		return nil, err
	}

	var E, parameterNum uint8
	if parameterNum = uint8(len(q.Parameters)); parameterNum != 0 {
		E = 1
	}
	if err := binary.Write(buf, binary.BigEndian, E<<6|parameterNum); err != nil {
		return nil, err
	}

	if E == 1 {
		if paraListBuf, err := q.Parameters.MarshalBinary(); err != nil {
			return nil, err
		} else {
			if _, err = buf.Write(paraListBuf); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func parseQosFlowDesc(buf *bytes.Buffer) (*QosFlowDesc, error) {
	var q QosFlowDesc
	// QFI
	if err := binary.Read(buf, binary.BigEndian, &q.QFI); err != nil {
		return nil, err
	}

	// Operation Code
	var OperationCodeOctet uint8
	if err := binary.Read(buf, binary.BigEndian, &OperationCodeOctet); err != nil {
		return nil, err
	}
	q.OperationCode = QosFlowOperationCode(OperationCodeOctet >> 5)

	// Parameter List
	var ParameterNumOctet uint8
	if err := binary.Read(buf, binary.BigEndian, &ParameterNumOctet); err != nil {
		return nil, err
	}
	if ParameterNumOctet != 0 {
		if paraList, err := parseQosFlowParameterList(buf, ParameterNumOctet&(1<<6-1)); err != nil {
			return nil, err
		} else {
			q.Parameters = paraList
		}
	}

	return &q, nil
}

type QosFlowParameterIdentifier uint8

const (
	ParameterIdentifier5QI               QosFlowParameterIdentifier = 0x01
	ParameterIdentifierGFBRUplink        QosFlowParameterIdentifier = 0x02
	ParameterIdentifierGFBRDownlink      QosFlowParameterIdentifier = 0x03
	ParameterIdentifierMFBRUplink        QosFlowParameterIdentifier = 0x04
	ParameterIdentifierMFBRDownlink      QosFlowParameterIdentifier = 0x05
	ParameterIdentifierAveragingWindow   QosFlowParameterIdentifier = 0x06
	ParameterIdentifierEPSBearerIdentity QosFlowParameterIdentifier = 0x07
)

type QosFlowParameterList []QosFlowParameter

func (l *QosFlowParameterList) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	for _, parameter := range *l {
		if err := binary.Write(buf, binary.BigEndian, parameter.Identifier()); err != nil {
			return nil, err
		}
		if parameterBuf, err := parameter.MarshalBinary(); err != nil {
			return nil, err
		} else {
			if err = binary.Write(buf, binary.BigEndian, uint8(len(parameterBuf))); err != nil {
				return nil, err
			}
			if _, err = buf.Write(parameterBuf); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func parseQosFlowParameterList(buf *bytes.Buffer, number uint8) (QosFlowParameterList, error) {
	paraList := make(QosFlowParameterList, 0)
	var parameterID QosFlowParameterIdentifier
	var parameterLen uint8

	for i := 0; i < int(number); i++ {
		if err := binary.Read(buf, binary.BigEndian, &parameterID); err != nil {
			return nil, err
		}
		if err := binary.Read(buf, binary.BigEndian, &parameterLen); err != nil {
			return nil, err
		}

		parameter := newQosFlowParameters(parameterID)

		if err := parameter.UnmarshalBinary(buf.Next(int(parameterLen))); err != nil {
			return nil, err
		}

		paraList = append(paraList, parameter)
	}

	return paraList, nil
}

type QosFlowParameter interface {
	Identifier() QosFlowParameterIdentifier
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}

func newQosFlowParameters(id QosFlowParameterIdentifier) QosFlowParameter {
	var parameter QosFlowParameter
	switch id {
	case ParameterIdentifier5QI:
		parameter = new(QosFlow5QI)
	case ParameterIdentifierGFBRUplink:
		parameter = &QosFlowGFBRUplink{}
	case ParameterIdentifierGFBRDownlink:
		parameter = &QosFlowGFBRDownlink{}
	case ParameterIdentifierMFBRUplink:
		parameter = &QosFlowMFBRUplink{}
	case ParameterIdentifierMFBRDownlink:
		parameter = &QosFlowMFBRDownlink{}
	case ParameterIdentifierAveragingWindow:
		parameter = &QosFlowAveragingWindow{}
	case ParameterIdentifierEPSBearerIdentity:
		parameter = &QosFlowEBI{}
	default:
		parameter = nil
	}
	return parameter
}

type QosFlow5QI struct {
	FiveQI uint8
}

func (q *QosFlow5QI) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifier5QI
}

func (q *QosFlow5QI) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// 5QI value
	if err := binary.Write(buf, binary.BigEndian, q.FiveQI); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (q *QosFlow5QI) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)

	// 5QI value
	if err := binary.Read(buf, binary.BigEndian, &q.FiveQI); err != nil {
		return err
	}

	return nil
}

type QosFlowBitRateUnit uint8

const (
	QosFlowBitRateUnit1Kbps QosFlowBitRateUnit = iota + 1
	QosFlowBitRateUnit4Kbps
	QosFlowBitRateUnit16Kbps
	QosFlowBitRateUnit64Kbps
	QosFlowBitRateUnit256Kbps
	QosFlowBitRateUnit1Mbps
	QosFlowBitRateUnit4Mbps
	QosFlowBitRateUnit16Mbps
	QosFlowBitRateUnit64Mbps
	QosFlowBitRateUnit256Mbps
	QosFlowBitRateUnit1Gbps
	QosFlowBitRateUnit4Gbps
	QosFlowBitRateUnit16Gbps
	QosFlowBitRateUnit64Gbps
	QosFlowBitRateUnit256Gbps
	QosFlowBitRateUnit1Tbps
	QosFlowBitRateUnit4Tbps
	QosFlowBitRateUnit16Tbps
	QosFlowBitRateUnit64Tbps
	QosFlowBitRateUnit256Tbps
	QosFlowBitRateUnit1Pbps
	QosFlowBitRateUnit4Pbps
	QosFlowBitRateUnit16Pbps
	QosFlowBitRateUnit64Pbps
	QosFlowBitRateUnit256Pbps
)

type qoSFlowBitRate struct {
	Unit  QosFlowBitRateUnit
	Value uint16
}

func (q *qoSFlowBitRate) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// BitRate Unit
	if err := binary.Write(buf, binary.BigEndian, uint8(q.Unit)); err != nil {
		return nil, err
	}

	// value
	if err := binary.Write(buf, binary.BigEndian, q.Value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *qoSFlowBitRate) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)

	// BitRate Unit
	if err := binary.Read(buf, binary.BigEndian, &q.Unit); err != nil {
		return err
	}

	// value
	if err := binary.Read(buf, binary.BigEndian, &q.Value); err != nil {
		return err
	}

	return nil
}

type QosFlowGFBRUplink qoSFlowBitRate

func (q *QosFlowGFBRUplink) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifierGFBRUplink
}

func (q *QosFlowGFBRUplink) MarshalBinary() ([]byte, error) {
	return (*qoSFlowBitRate)(q).MarshalBinary()
}

func (q *QosFlowGFBRUplink) UnmarshalBinary(b []byte) error {
	return (*qoSFlowBitRate)(q).UnmarshalBinary(b)
}

type QosFlowGFBRDownlink qoSFlowBitRate

func (q *QosFlowGFBRDownlink) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifierGFBRDownlink
}

func (q *QosFlowGFBRDownlink) MarshalBinary() ([]byte, error) {
	return (*qoSFlowBitRate)(q).MarshalBinary()
}

func (q *QosFlowGFBRDownlink) UnmarshalBinary(b []byte) error {
	return (*qoSFlowBitRate)(q).UnmarshalBinary(b)
}

type QosFlowMFBRUplink qoSFlowBitRate

func (q *QosFlowMFBRUplink) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifierMFBRUplink
}

func (q *QosFlowMFBRUplink) MarshalBinary() ([]byte, error) {
	return (*qoSFlowBitRate)(q).MarshalBinary()
}

func (q *QosFlowMFBRUplink) UnmarshalBinary(b []byte) error {
	return (*qoSFlowBitRate)(q).UnmarshalBinary(b)
}

type QosFlowMFBRDownlink qoSFlowBitRate

func (q *QosFlowMFBRDownlink) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifierMFBRDownlink
}

func (q *QosFlowMFBRDownlink) MarshalBinary() ([]byte, error) {
	return (*qoSFlowBitRate)(q).MarshalBinary()
}

func (q *QosFlowMFBRDownlink) UnmarshalBinary(b []byte) error {
	return (*qoSFlowBitRate)(q).UnmarshalBinary(b)
}

type QosFlowAveragingWindow struct {
	AverageWindow uint16
}

func (q *QosFlowAveragingWindow) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifierAveragingWindow
}

func (q *QosFlowAveragingWindow) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// AverageWindow value in milliseconds
	if err := binary.Write(buf, binary.BigEndian, q.AverageWindow); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *QosFlowAveragingWindow) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)

	// AverageWindow value in milliseconds
	if err := binary.Read(buf, binary.BigEndian, &q.AverageWindow); err != nil {
		return err
	}

	return nil
}

type QosFlowEBI struct {
	EBI uint8
}

func (q *QosFlowEBI) Identifier() QosFlowParameterIdentifier {
	return ParameterIdentifierEPSBearerIdentity
}

func (q *QosFlowEBI) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// EBI
	if err := binary.Write(buf, binary.BigEndian, q.EBI); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *QosFlowEBI) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)

	// EBI
	if err := binary.Read(buf, binary.BigEndian, &q.EBI); err != nil {
		return err
	}

	return nil
}
