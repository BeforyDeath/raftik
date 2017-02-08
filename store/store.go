package store

import "github.com/garyburd/redigo/redis"

type Store struct {
	conn redis.Conn
}

func NewStore(network, address string) (Store, error) {
	conn, err := redis.Dial(network, address)
	s := Store{
		conn: conn,
	}
	return s, err
}

func (s Store) Close() error {
	return s.conn.Close()
}

func (s Store) AddUUID(uuid string) error {
	_, err := s.conn.Do("RPUSH", "node:list", uuid)
	return err
}

func (s Store) GetNodes() ([]string, error) {
	res, err := redis.Strings(s.conn.Do("LRANGE", "node:list", "0", "-1"))
	return res, err
}
func (s Store) Ping(uuid string) (int, error) {
	res, err := redis.Int(s.conn.Do("INCR", uuid+":ping"))
	return res, err
}
func (s Store) PingMe(uuid string) error {
	_, err := s.conn.Do("SET", uuid+":ping", 0)
	return err
}

func (s Store) Erase(uuid string) error {
	s.conn.Do("LREM", "node:list", 0, uuid)
	s.conn.Do("DEL", uuid+":Vote:request")
	s.conn.Do("DEL", uuid+":Vote:reply")
	s.conn.Do("DEL", uuid+":Append:request")
	s.conn.Do("DEL", uuid+":Append:reply")
	s.conn.Do("DEL", uuid+":ping")
	return nil //fixme
}

func (s Store) SetVoteRequest(uuid string, b []byte) error {
	_, err := s.conn.Do("RPUSH", uuid+":Vote:request", b)
	return err
}
func (s Store) GetVoteReply(uuid string) ([]string, error) {
	res, err := redis.Strings(s.conn.Do("LRANGE", uuid+":Vote:reply", "0", "-1"))
	s.conn.Do("DEL", uuid+":Vote:reply")
	return res, err
}

func (s Store) SetVoteReply(uuid string, b []byte) error {
	_, err := s.conn.Do("RPUSH", uuid+":Vote:reply", b)
	return err
}

func (s Store) GetVoteRequest(uuid string) ([]string, error) {
	res, err := redis.Strings(s.conn.Do("LRANGE", uuid+":Vote:request", "0", "-1"))
	s.conn.Do("DEL", uuid+":Vote:request")
	return res, err
}

func (s Store) SetAppendRequest(uuid string, b []byte) error {
	_, err := s.conn.Do("RPUSH", uuid+":Append:request", b)
	//todo msg send counter
	s.conn.Do("INCR", "msg:counter")
	return err
}
func (s Store) GetAppendRequest(uuid string) ([]byte, error) {
	res, err := redis.Bytes(s.conn.Do("RPOP", uuid+":Append:request"))
	return res, err
}

func (s Store) SetError(msg string) error {
	_, err := s.conn.Do("RPUSH", "Errors", msg)
	return err
}
