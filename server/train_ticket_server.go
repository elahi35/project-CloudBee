package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"
)

// Protobuf definitions embedded directly in Golang code
type TicketRequest struct {
	From          string `json:"from"`
	To            string `json:"to"`
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
	UserEmail     string `json:"user_email"`
}

type User struct {
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
	UserEmail     string `json:"user_email"`
}

type Receipt struct {
	From          string  `json:"from"`
	To            string  `json:"to"`
	UserFirstName string  `json:"user_first_name"`
	UserLastName  string  `json:"user_last_name"`
	UserEmail     string  `json:"user_email"`
	PricePaid     float64 `json:"price_paid"`
	Seat          string  `json:"seat"`
	Section       string  `json:"section"`
}

type SectionRequest struct {
	Section string `json:"section"`
}

type UsersBySection struct {
	Users   []User `json:"users"`
	Section string `json:"section"`
}

type RemoveUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UserSeatRequest struct {
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
	NewSeat       string `json:"new_seat"`
}

type ModifyUserSeatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	NewSeat string `json:"new_seat"`
	Section string `json:"section"`
}

type trainServer struct {
	Users     map[string]User
	UserSeats map[string]string
	SectionA  map[string]string
	SectionB  map[string]string
}

func (s *trainServer) PurchaseTicket(ctx context.Context, req *TicketRequest) (*Receipt, error) {
	seat, section := s.allocateSeat(req)

	userKey := getUserKey(req)
	s.Users[userKey] = User{
		UserFirstName: req.UserFirstName,
		UserLastName:  req.UserLastName,
		UserEmail:     req.UserEmail,
	}

	return &Receipt{
		From:          req.From,
		To:            req.To,
		UserFirstName: req.UserFirstName,
		UserLastName:  req.UserLastName,
		UserEmail:     req.UserEmail,
		PricePaid:     20.0,
		Seat:          seat,
		Section:       section,
	}, nil
}

func (s *trainServer) GetReceiptDetails(ctx context.Context, req *User) (*Receipt, error) {
	userKey := getUserKey(req)
	user, ok := s.Users[userKey]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}

	seat, ok := s.UserSeats[userKey]
	if !ok {
		return nil, fmt.Errorf("seat not found for user")
	}

	return &Receipt{
		From:          "London",
		To:            "France",
		UserFirstName: user.UserFirstName,
		UserLastName:  user.UserLastName,
		UserEmail:     user.UserEmail,
		PricePaid:     20.0,
		Seat:          seat,
	}, nil
}

func (s *trainServer) GetUsersBySection(ctx context.Context, req *SectionRequest) (*UsersBySection, error) {
	var users []User
	var sectionMap map[string]string

	switch req.Section {
	case "A":
		sectionMap = s.SectionA
	case "B":
		sectionMap = s.SectionB
	default:
		return nil, fmt.Errorf("invalid section")
	}

	for userKey,_ := range sectionMap {
		names := strings.Split(userKey, "-")
		users = append(users, User{
			UserFirstName: names[0],
			UserLastName:  names[1],
			UserEmail:     s.Users[userKey].UserEmail,
		})
	}

	return &UsersBySection{
		Users:   users,
		Section: req.Section,
	}, nil
}

func (s *trainServer) RemoveUser(ctx context.Context, req *User) (*RemoveUserResponse, error) {
	userKey := getUserKey(req)

	if _, ok := s.Users[userKey]; !ok {
		return &RemoveUserResponse{
			Success: false,
			Message: "User not found",
		}, nil
	}

	// Remove user from Users map
	delete(s.Users, userKey)

	// Remove user from SectionA or SectionB
	if seat, ok := s.UserSeats[userKey]; ok {
		delete(s.UserSeats, userKey)
		switch seat {
		case "A":
			delete(s.SectionA, userKey)
		case "B":
			delete(s.SectionB, userKey)
		}
	}

	return &RemoveUserResponse{
		Success: true,
		Message: "User removed successfully",
	}, nil
}

func (s *trainServer) ModifyUserSeat(ctx context.Context, req *UserSeatRequest) (*ModifyUserSeatResponse, error) {
	userKey := getUserKey(req)

	if _, ok := s.Users[userKey]; !ok {
		return &ModifyUserSeatResponse{
			Success: false,
			Message: "User not found",
		}, nil
	}

	if _, ok := s.UserSeats[userKey]; !ok {
		return &ModifyUserSeatResponse{
			Success: false,
			Message: "Seat not found for user",
		}, nil
	}

	// Remove user from the current seat
	seat := s.UserSeats[userKey]
	delete(s.UserSeats, userKey)

	// Remove user from the current section
	switch seat {
	case "A":
		delete(s.SectionA, userKey)
	case "B":
		delete(s.SectionB, userKey)
	}

	// Allocate a new seat
	newSeat, newSection := s.allocateSeat(&TicketRequest{
		UserFirstName: req.UserFirstName,
		UserLastName:  req.UserLastName,
		UserEmail:     s.Users[userKey].UserEmail,
	})

	return &ModifyUserSeatResponse{
		Success: true,
		Message: fmt.Sprintf("Seat modified for %s", req.UserFirstName),
		NewSeat: newSeat,
		Section: newSection,
	}, nil
}

func (s *trainServer) allocateSeat(req *TicketRequest) (string, string) {
	section := "A"
	if len(s.UserSeats)%2 == 1 {
		section = "B"
	}

	seat := fmt.Sprintf("%s-%s", req.UserFirstName, req.UserLastName)
	s.UserSeats[seat] = section

	switch section {
	case "A":
		s.SectionA[seat] = section
	case "B":
		s.SectionB[seat] = section
	}

	return seat, section
}

func getUserKey(req interface{}) string {
	switch v := req.(type) {
	case *TicketRequest:
		return fmt.Sprintf("%s-%s", v.UserFirstName, v.UserLastName)
	case *User:
		return fmt.Sprintf("%s-%s", v.UserFirstName, v.UserLastName)
	case *UserSeatRequest:
		return fmt.Sprintf("%s-%s", v.UserFirstName, v.UserLastName)
	default:
		return ""
	}
}

func main() {
	server := &trainServer{
		Users:     make(map[string]User),
		UserSeats: make(map[string]string),
		SectionA:  make(map[string]string),
		SectionB:  make(map[string]string),
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	RegisterTrainServiceServer(s, server)
	log.Println("Server started on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func RegisterTrainServiceServer(s *grpc.Server, server *trainServer) {
	fmt.Println("unimplemented")
}