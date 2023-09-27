package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type SploaderService struct {
	db *sql.DB
}


func NewSploaderService(db *sql.DB) *SploaderService {
	return &SploaderService{
		db: db,
	}
}


func (s *SploaderService) DetermineApplicationType(applicationId string) (string,error) {

	var subscriptionType string

	query := `SELECT subscriptionType from applications where id = ?`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, query)

	if err != nil {
		return subscriptionType, err
	}

	defer stmt.Close()

	
	result, err := stmt.QueryContext(ctx, applicationId)

	if err != nil {
		return subscriptionType, err
	}

	err = result.Scan(&subscriptionType)

	log.Printf("Application type db result %v", result)

	if err != nil {
		return subscriptionType, err
	}

	return subscriptionType, nil
}


func (s *SploaderService) CalculateApplicationStorage(applicationId string) (bool, error) {

	query := `SELECT SUM(size) FROM uploads WHERE application_id = ?`
	

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		stmt, err := s.db.PrepareContext(ctx, query)

		if err != nil {
			return false, err
		}

		defer stmt.Close()


		result, err := stmt.ExecContext(ctx, applicationId)

		if err != nil {
			return false, err
		}

		log.Printf("Wrote new upload to db with result %v", result)

		return true, nil
}

