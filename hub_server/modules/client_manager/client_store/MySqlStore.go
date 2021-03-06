package client_store

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"strconv"
	"time"
	"whub/hub_server/client"
)

const (
	SQLTimeFormat = "2006-01-02 15:04:05"
	// TODO only for test
	SQLServer   = "192.168.0.164:3307"
	SQLUserName = "root"
	SQLPassword = "Lxr000518!"
	SQLDBName   = "wr_test"

	RedisAddr = "192.168.0.132:6379"
	RedisPass = "19950416"
)

type DClient struct {
	ID          string `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string
	PScope      int
	CKey        string
	CType       int
}

func (d *DClient) toClient() *client.Client {
	return client.NewClient(d.ID, d.Description, d.CType, d.CKey, d.PScope)
}

type MySqlStore struct {
	db *gorm.DB
}

func NewMySqlClientStore() *MySqlStore {
	return &MySqlStore{}
}

func (s *MySqlStore) clientToDClient(client *client.Client) *DClient {
	return &DClient{
		ID:          client.Id(),
		Description: client.Description(),
		CType:       client.CType(),
		CKey:        client.CKey(),
		PScope:      client.PScope(),
	}
}

func (s *MySqlStore) Init(fullDBUri, username, password, dbname string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, fullDBUri, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	s.db = db
	return s.db.AutoMigrate(&DClient{})
}

func (s *MySqlStore) Get(id string) (*client.Client, error) {
	queryHolder := &DClient{ID: id}
	result := s.db.First(queryHolder)
	if result.Error != nil {
		return nil, result.Error
	}
	return queryHolder.toClient(), nil
}

func (s *MySqlStore) Create(client *client.Client) error {
	return s.db.Create(s.clientToDClient(client)).Error
}

func (s *MySqlStore) Update(client *client.Client) error {
	return s.db.Updates(s.clientToDClient(client)).Error
}

func (s *MySqlStore) Has(id string) (bool, error) {
	c, e := s.Get(id)
	if e != nil {
		return false, e
	}
	return c != nil, e
}

func (s *MySqlStore) Delete(id string) error {
	return s.db.Delete(&DClient{ID: id}).Error
}

func (s *MySqlStore) GetAll() ([]*client.Client, error) {
	return s.batchFindOperations(s.db.Where("1 = ?", "1"))
}

func (s *MySqlStore) Find(query *DClientQuery) ([]*client.Client, error) {
	return s.batchFindOperations(s.buildQueryTx(query))
}

func (s *MySqlStore) buildQueryTx(query *DClientQuery) *gorm.DB {
	var clauses []string
	if query.cType > -1 {
		clauses = append(clauses, "c_type = ?", strconv.Itoa(query.cType))
	}
	if !(query.createdBefore.IsZero() || query.createdAfter.IsZero()) {
		clauses = append(clauses, "created_at BETWEEN ? AND ?", query.createdAfter.Format(SQLTimeFormat), query.createdBefore.Format(SQLTimeFormat))
	} else if !query.createdBefore.IsZero() {
		clauses = append(clauses, "created_at > ?", query.createdAfter.Format(SQLTimeFormat))
	} else if !query.createdBefore.IsZero() {
		clauses = append(clauses, "created_at < ?", query.createdBefore.Format(SQLTimeFormat))
	}
	tx := s.db.Where(clauses).Limit(query.limit)
	if query.limit > 0 {
		return tx.Limit(query.limit)
	}
	return tx
}

func (s *MySqlStore) batchFindOperations(tx *gorm.DB) ([]*client.Client, error) {
	var allClients []*DClient
	if err := tx.Find(&allClients).Error; err != nil {
		return nil, err
	}
	transformedClients := make([]*client.Client, len(allClients), len(allClients))
	for i, c := range allClients {
		transformedClients[i] = c.toClient()
	}
	return transformedClients, nil
}

func (s *MySqlStore) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
