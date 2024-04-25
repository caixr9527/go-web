package rpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"io"
	"log"
	"net"
	"reflect"
	"time"
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

type ProtobufSerializer struct {
}

func (c ProtobufSerializer) Deserialize(data []byte, target any) error {
	message := target.(proto.Message)
	return proto.Unmarshal(data, message)
}

func (c ProtobufSerializer) Serialize(data any) ([]byte, error) {
	marshal, err := proto.Marshal(data.(proto.Message))
	if err != nil {
		return nil, err
	}
	return marshal, err
}

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
		// todo
	}
	headers := make([]byte, 17)
	headers[0] = MagicNumber
	headers[1] = Version
	headers[6] = byte(msgResponse)
	headers[7] = byte(rsp.CompressType)
	headers[8] = byte(rsp.SerializerType)
	binary.BigEndian.PutUint64(headers[9:], uint64(rsp.RequestId))
	se := loadSerializer(rsp.SerializerType)
	var body []byte
	var err error
	if rsp.SerializerType == ProtoBuff {
		pRsp := &Response{}
		pRsp.SerializeType = int32(rsp.SerializerType)
		pRsp.CompressType = int32(rsp.CompressType)
		pRsp.Code = int32(rsp.Code)
		pRsp.Msg = rsp.Msg
		pRsp.RequestId = rsp.RequestId
		m := make(map[string]any)
		marshal, _ := json.Marshal(rsp.Data)
		_ = json.Unmarshal(marshal, &m)
		value, err := structpb.NewStruct(m)
		// todo
		log.Println(err)
		pRsp.Data = structpb.NewStructValue(value)
		body, _ = se.Serialize(pRsp)
	} else {
		body, err = se.Serialize(rsp)
	}

	if err != nil {
		return err
	}
	com := loadCompress(rsp.CompressType)
	body, err = com.Compress(body)
	if err != nil {
		return err
	}
	fullLen := 17 + len(body)
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen))

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

	defer func() {
		if err := recover(); err != nil {
			// todo
			log.Println(err)
			conn.conn.Close()
		}
	}()

	msg, err := decodeFrame(conn.conn)
	if err != nil {
		rsp := &MsgRpcResponse{}
		rsp.Code = 500
		rsp.Msg = err.Error()
		conn.rspChan <- rsp
		return
	}
	if msg.Header.MessageType == msgRequest {
		if msg.Header.SerializerType == ProtoBuff {
			req := msg.Data.(*Request)
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

			args := make([]reflect.Value, len(req.Args))
			for i := range req.Args {
				of := reflect.ValueOf(req.Args[i].AsInterface())
				of = of.Convert(method.Type().In(i))
				args[i] = of
			}
			result := method.Call(args)

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
		} else {
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

func decodeFrame(conn net.Conn) (*MsgRpcMessage, error) {
	headers := make([]byte, 17)
	_, err := io.ReadFull(conn, headers)
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

	msg := &MsgRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn
	msg.Header.Version = version
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(messageType)
	msg.Header.CompressType = CompressType(compressType)
	msg.Header.SerializerType = SerializerType(seType)
	msg.Header.RequestId = requestId

	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)

	_, err = io.ReadFull(conn, body)
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
		var req any
		if SerializerType(seType) == ProtoBuff {
			req = &Request{}
		} else {
			req = &MsgRpcRequest{}
		}
		err := serializer.Deserialize(body, req)
		if err != nil {
			return nil, err
		}
		msg.Data = req
		return msg, nil
	}
	if MessageType(messageType) == msgResponse {
		var rsp any
		if SerializerType(seType) == ProtoBuff {
			rsp = &Response{}
		} else {
			rsp = &MsgRpcResponse{}
		}

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
		return ProtobufSerializer{}
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

type RpcClient interface {
	Connect() error
	Invoke(context context.Context, serviceName string, methodName string, args []any) (any, error)
	Close() error
}
type TcpClient struct {
	conn   net.Conn
	option TcpClientOption
}

type TcpClientOption struct {
	Retries           int
	ConnectionTimeout time.Duration
	SerializerType    SerializerType
	CompressType      CompressType
	Host              string
	Port              int
}

var DefaultOption = TcpClientOption{
	Retries:           3,
	ConnectionTimeout: 5 * time.Second,
	SerializerType:    Gob,
	CompressType:      Gzip,
	Host:              "127.0.0.1",
	Port:              9222,
}

func NewTcpClient(option TcpClientOption) *TcpClient {
	return &TcpClient{option: option}
}

func (c *TcpClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.option.Host, c.option.Port)
	conn, err := net.DialTimeout("tcp", addr, c.option.ConnectionTimeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *TcpClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *TcpClient) Invoke(context context.Context, serviceName string, methodName string, args []any) (any, error) {
	req := &MsgRpcRequest{}
	// todo uuid
	req.RequestId = 1
	req.ServiceName = serviceName
	req.MethodName = methodName
	req.Args = args

	headers := make([]byte, 17)
	headers[0] = MagicNumber
	headers[1] = Version
	headers[6] = byte(msgRequest)
	headers[7] = byte(c.option.CompressType)
	headers[8] = byte(c.option.SerializerType)
	binary.BigEndian.PutUint64(headers[9:], uint64(req.RequestId))
	serializer := loadSerializer(c.option.SerializerType)
	if serializer == nil {
		return nil, errors.New("serializer method not found")
	}
	var body []byte
	var err error
	if c.option.SerializerType == ProtoBuff {
		pReq := &Request{}
		pReq.RequestId = 1
		pReq.ServiceName = serviceName
		pReq.MethodName = methodName
		listValue, err := structpb.NewList(args)
		if err != nil {
			return nil, err
		}
		pReq.Args = listValue.Values
		body, err = serializer.Serialize(pReq)
	} else {
		body, err = serializer.Serialize(req)

	}

	if err != nil {
		return nil, err
	}
	compress := loadCompress(c.option.CompressType)
	if compress == nil {
		return nil, errors.New("compress method not found")
	}
	body, err = compress.Compress(body)
	if err != nil {
		return nil, err
	}
	fullLen := 17 + len(body)
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen))
	_, err = c.conn.Write(headers[:])
	if err != nil {
		return nil, err
	}

	_, err = c.conn.Write(body[:])
	if err != nil {
		return nil, err
	}
	rspChan := make(chan *MsgRpcResponse)
	go c.readHandler(rspChan)
	rsp := <-rspChan
	return rsp, nil
}

func (c *TcpClient) readHandler(rspChan chan *MsgRpcResponse) {
	defer func() {
		if err := recover(); err != nil {
			//todo
			log.Println(err)
			c.conn.Close()
		}
	}()
	for {
		msg, err := decodeFrame(c.conn)
		if err != nil {
			//todo
			log.Println("not msg")
			rsp := &MsgRpcResponse{}
			rsp.Code = 500
			rsp.Msg = err.Error()
			rspChan <- rsp
			return
		}
		if msg.Header.MessageType == msgResponse {
			if msg.Header.SerializerType == ProtoBuff {
				rsp := msg.Data.(*Response)
				asInterface := rsp.Data.AsInterface()
				marshal, _ := json.Marshal(asInterface)
				rspl := &MsgRpcResponse{}
				json.Unmarshal(marshal, rspl)
				rspChan <- rspl
			} else {
				rsp := msg.Data.(*MsgRpcResponse)
				rspChan <- rsp
			}
			return
		}
	}
}

type TcpClientProxy struct {
	client *TcpClient
	option TcpClientOption
}

func NewTcpClientProxy(option TcpClientOption) *TcpClientProxy {
	return &TcpClientProxy{option: option}
}

// todo args换一种格式,map
func (p *TcpClientProxy) Call(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	client := NewTcpClient(p.option)
	p.client = client
	err := client.Connect()
	if err != nil {
		return nil, err
	}
	for i := 0; i < p.option.Retries; i++ {
		result, err := client.Invoke(ctx, serviceName, methodName, args)
		if err != nil {
			if i >= p.option.Retries-1 {
				//todo
				log.Println(errors.New("already retry all time"))
				client.Close()
				return nil, err
			}
			//todo sleep一会儿
			continue
		}
		client.Close()
		return result, nil
	}
	return nil, errors.New("retry time is 0")
}
