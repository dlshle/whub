package client_store

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
	"wsdk/relay_server/client"
)

type ClientDBO struct {
	ID          uint   `gorm:"primaryKey"`
	AliasId     string `gorm:"unique"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string
	PScope      int
	CKey        string
	CType       int
}

type MySqlClientStore struct {
	db *gorm.DB
}

func (s *MySqlClientStore) dbo2bo(dbo *ClientDBO) *client.Client {
	return client.NewClient(dbo.AliasId, dbo.Description, dbo.CType, dbo.CKey, dbo.PScope)
}

func (s *MySqlClientStore) Init(fullServerUri, username, password, dbname string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, fullServerUri, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	s.db = db
	s.db.AutoMigrate(&ClientDBO{})
	return nil
}

func (s *MySqlClientStore) Get(id string) (*client.Client, error) {
	queryHolder := ClientDBO{AliasId: id}
	result := s.db.First(&queryHolder)
	if result.Error != nil {
		return nil, result.Error
	}
	return s.dbo2bo(&queryHolder), nil
}

func (s *MySqlClientStore) Put(id string, client *client.Client) error {
	// TODO
}

func (s *MySqlClientStore) Has(id string) (bool, error) {
	// TODO
}

func (s *MySqlClientStore) Delete(id string) error {
	// TODO
}

func (s *MySqlClientStore) Find(query IQuery) (*client.Client, error) {
	// TODO
}
