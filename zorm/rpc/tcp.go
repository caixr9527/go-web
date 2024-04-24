package rpc

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
)

type Serializer interface {
	Serialize(data any) ([]byte, error)
	Deserialize(data []byte, target any) error
}

type GobSerializer struct {
}

func (g GobSerializer) Serialize(data any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (g GobSerializer) Deserialize(data []byte, target any) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	return decoder.Decode(target)
}

type SerializerType byte

const (
	Gob SerializerType = iota
	ProtoBuff
)

type Compress interface {
	Compress([]byte) ([]byte, error)
	UnCompress([]byte) ([]byte, error)
}

type CompressType byte

const (
	Gzip CompressType = iota
)

type GzipCompress struct {
}

func (g GzipCompress) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GzipCompress) UnCompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const MagicNumber byte = 0x1d
const Version = 0x01

type MessageType byte

const (
	msgRequest MessageType = iota
	msgResponse
	msgPing
	msgPong
)

type Header struct {
	MagicNumber    byte
	Version        byte
	FullLength     int32
	MessageType    MessageType
	CompressType   CompressType
	SerializerType SerializerType
	RequestId      int64
}

type MsgRpcMessage struct {
	Header *Header
	Data   any
}

type MsgRpcRequest struct {
	RequestId   int64
	ServiceName string
	MethodName  string
	Args        []any
}

type MsgRpcResponse struct {
	RequestId      int64
	Code           int16
	Msg            string
	CompressType   CompressType
	SerializerType SerializerType
	Data           any
}

type MsgRpcServer interface {
	Register(name string, service interface{})
	Run()
	Stop()
}

type MsgTcpServer struct {
	listen     net.Listener
	Host       string
	Port       int
	NetWork    string
	serviceMap map[string]any
}

func NewTcpServer(addr string) (*MsgTcpServer, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &MsgTcpServer{
		listen:     listen,
		serviceMap: make(map[string]any),
	}, nil
}

func (s *MsgTcpServer) Register(name string, service interface{}) {
	t := reflect.TypeOf(service)
	if t.Kind() != reflect.Pointer {
		panic("service must be pointer")
	}
	s.serviceMap[name] = service
}

type MsgTcpConn struct {
	conn    net.Conn
	rspChan chan *MsgRpcResponse
}

func (c MsgTcpConn) Send(rsp *MsgRpcResponse) error {
	if rsp.Code != 200 {

	}
	headers := make([]byte, 17)
	headers[0] = MagicNumber
	headers[1] = Version
	headers[6] = byte(msgResponse)
	headers[7] = byte(rsp.CompressType)
	headers[8] = byte(rsp.SerializerType)
	binary.BigEndian.PutUint64(headers[9:], uint64(rsp.RequestId))
	se := loadSerializer(rsp.SerializerType)
	body, err := se.Serialize(rsp.Data)
	if err != nil {
		return err
	}
	com := loadCompress(rsp.CompressType)
	body, err = com.Compress(body)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(headers[:])
	if err != nil {
		return err
	}
	_, err = c.conn.Write(body[:])
	if err != nil {
		return err
	}
	return nil
}

func (s *MsgTcpServer) Run() {
	for {
		conn, err := s.listen.Accept()
		if err != nil {
			// Todo
			log.Println(err)
			continue
		}
		msgConn := &MsgTcpConn{conn: conn, rspChan: make(chan *MsgRpcResponse, 1)}
		go s.readHandler(msgConn)
		go s.writeHandler(msgConn)
	}
}

func (s *MsgTcpServer) Stop() {
	_ = s.listen.Close()
}

func (s *MsgTcpServer) readHandler(conn *MsgTcpConn) {
	msg, err := s.decodeFrame(conn)
	if err != nil {
		rsp := &MsgRpcResponse{}
		rsp.Code = 500
		rsp.Msg = err.Error()
		conn.rspChan <- rsp
		return
	}
	if msg.Header.MessageType == msgRequest {
		req := msg.Data.(*MsgRpcRequest)
		rsp := &MsgRpcResponse{RequestId: req.RequestId}
		rsp.SerializerType = msg.Header.SerializerType
		rsp.CompressType = msg.Header.CompressType
		serviceName := req.ServiceName
		service, ok := s.serviceMap[serviceName]
		if !ok {
			rsp := &MsgRpcResponse{}
			rsp.Code = 500
			rsp.Msg = fmt.Sprintf("service: [%s] not found", serviceName)
			conn.rspChan <- rsp
			return
		}
		methodName := req.MethodName
		method := reflect.ValueOf(service).MethodByName(methodName)
		if method.IsNil() {
			rsp := &MsgRpcResponse{}
			rsp.Code = 500
			rsp.Msg = fmt.Sprintf("service: [%s] method: [%s] not found", serviceName, methodName)
			conn.rspChan <- rsp
			return
		}
		args := req.Args
		var valuesArg []reflect.Value
		for _, v := range args {
			valuesArg = append(valuesArg, reflect.ValueOf(v))
		}
		result := method.Call(valuesArg)
		results := make([]any, len(result))
		for i, v := range result {
			results[i] = v.Interface()
		}
		err, ok := results[len(result)-1].(error)
		if ok {
			rsp.Code = 500
			rsp.Msg = err.Error()
			conn.rspChan <- rsp
			return
		}
		rsp.Code = 200
		rsp.Data = results[0]
		conn.rspChan <- rsp

	}
}

func (s *MsgTcpServer) writeHandler(conn *MsgTcpConn) {
	select {
	case rsp := <-conn.rspChan:
		defer conn.conn.Close()
		err := conn.Send(rsp)
		if err != nil {
			// todo
			log.Println(err)
		}

	}
}

func (s *MsgTcpServer) decodeFrame(conn *MsgTcpConn) (*MsgRpcMessage, error) {
	headers := make([]byte, 17)
	_, err := io.ReadFull(conn.conn, headers)
	if err != nil {
		return nil, err
	}
	mn := headers[0]
	if mn != MagicNumber {
		return nil, errors.New("magic number error")
	}
	version := headers[1]
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6]))
	messageType := headers[6]
	compressType := headers[7]
	seType := headers[8]
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))

	msg := &MsgRpcMessage{}
	msg.Header.MagicNumber = mn
	msg.Header.Version = version
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(messageType)
	msg.Header.CompressType = CompressType(compressType)
	msg.Header.SerializerType = SerializerType(seType)
	msg.Header.RequestId = requestId

	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)

	_, err = io.ReadFull(conn.conn, body)
	if err != nil {
		return nil, err
	}

	compress := loadCompress(CompressType(compressType))
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.UnCompress(body)
	if err != nil {
		return nil, err
	}
	serializer := loadSerializer(SerializerType(seType))
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	if MessageType(messageType) == msgRequest {
		req := &MsgRpcRequest{}
		err := serializer.Deserialize(body, req)
		if err != nil {
			return nil, err
		}
		msg.Data = req
		return msg, nil
	}
	if MessageType(messageType) == msgResponse {
		rsp := &MsgRpcResponse{}
		err := serializer.Deserialize(body, rsp)
		if err != nil {
			return nil, err
		}
		msg.Data = rsp
		return msg, nil
	}
	return nil, errors.New("no message type")
}

func loadSerializer(serializerType SerializerType) Serializer {
	switch serializerType {
	case Gob:
		return GobSerializer{}
	case ProtoBuff:
		// todo
		return nil
	}
	return nil
}

func loadCompress(compressType CompressType) Compress {
	switch compressType {
	case Gzip:
		return GzipCompress{}
	}
	return nil
}
