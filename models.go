package main

import "time"

//клиенты
type Client struct {
	ID            int
	Name          string
	ContactPerson string
	Email         string
	Phone         string
	Industry      string
	CreatedAt     time.Time
}

//проект 
type Engagement struct {
	ID         int
	ClientID   int
	ClientName string
	Name       string
	Type       string
	Status     string
	StartDate  string
	EndDate    string
	Scope      string
	Pentester  string
	Findings   []Finding
}

//найденные уяз
type Finding struct {
	ID             int
	EngagementID   int
	EngagementName string
	Title          string
	Description    string
	Severity       string
	Cvss           float64
	CveID          string
	Status         string
	Recommendation string
}
