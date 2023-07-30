package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/joho/godotenv"
	pb "github.com/yokawasa/grpc-bookstore/proto"
)

type server struct {
	pb.UnimplementedBookstoreServer
}

// mongo setting
var db *mongo.Client
var bookDB *mongo.Collection
var mongoCtx context.Context

// book interface
type BookInterface struct {
	BookID   string `bson:bookId`
	BookName string `bson:bookName`
	Category string `bson:category`
	Author   string `bson:author`
}

// Book to pb response
func BookToProto(data *BookInterface) *pb.Book {
	return &pb.Book{
		BookID:   data.BookID,
		BookName: data.BookName,
		Category: data.Category,
		Author:   data.Author,
	}
}

// Create new book
func (s *server) PostBook(ctx context.Context, req *pb.BookRequest) (*pb.BookResponse, error) {
	// Get the request
	book := req.GetBook()

	data := BookInterface{
		BookID:   book.GetBookID(),
		BookName: book.GetBookName(),
		Category: book.GetCategory(),
		Author:   book.GetAuthor(),
	}

	// Insert the data into the database
	res, err := bookDB.InsertOne(mongoCtx, data)
	if err != nil {
		return nil,
			status.Errorf(codes.Internal, fmt.Sprintf(" Internal Error: %v", err))
	}

	fmt.Printf("Successfully inserted NEW Book into book collection!, %v \n", res)

	return &pb.BookResponse{Book: BookToProto(&data)}, nil
}

// Read book by ID of the book
func (s *server) GetBook(ctx context.Context, req *pb.GetBookReq) (*pb.BookResponse, error) {
	// Get ID of the book
	bookID := req.GetId()

	res := bookDB.FindOne(ctx, bson.M{"bookid": bookID})
	data := &BookInterface{}

	// decode and Check for error
	if err := res.Decode(data); err != nil {
		return nil,
			status.Errorf(codes.NotFound, fmt.Sprintf("Cannot found book with the ID: %v", err))
	}
	fmt.Println("Get book result", data)
	return &pb.BookResponse{Book: BookToProto(data)}, nil
}

// Update book
func (s *server) UpdateBook(ctx context.Context, req *pb.BookRequest) (*pb.BookResponse, error) {
	// Get the request
	book := req.GetBook()

	data := bson.M{
		"bookid":   book.BookID,
		"bookname": book.BookName,
		"category": book.Category,
		"author":   book.Author,
	}
	// insert the changes
	bookDB.FindOneAndUpdate(
		ctx,
		bson.M{"bookid": book.BookID},
		bson.M{"$set": data})

	fmt.Println("the decode result is:", book)

	return &pb.BookResponse{
		Book: &pb.Book{
			BookID:   req.GetBook().BookID,
			BookName: req.GetBook().BookName,
			Category: req.GetBook().Category,
			Author:   req.GetBook().Author,
		},
	}, nil
}

//
func (s *server) DeleteBook(ctx context.Context, req *pb.GetBookReq) (*pb.DeleteBookRes, error) {
	// Get ID of the book
	bookID := req.GetId()

	res, err := bookDB.DeleteOne(ctx, bson.M{"bookid": bookID})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Book deleted: ", res.DeletedCount)

	return &pb.DeleteBookRes{Deleted: res.DeletedCount}, nil
}

// Get all books in the Collection
func (s *server) GetAllBooks(ctx context.Context, req *pb.GetAllReq) (*pb.GetAllResponse, error) {
	fmt.Println("\n list of all book start stream")

	res, err := bookDB.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Unknown Internal Error: %v", err))
	}

	defer res.Close(context.Background())

	var books = []*BookInterface{}

	for res.Next(context.Background()) {
		var data = &BookInterface{}
		if err := res.Decode(data); err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Cannot decoding data: %v", err))
		}
		books = append(books, data)
	}
	if err = res.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Unknown Internal Error: %v", err))
	}
	var pbbooks = []*pb.Book{}
	for _, data := range books {
		fmt.Println(data)
		pbbooks = append(pbbooks, &pb.Book{BookID: data.BookID, BookName: data.BookName,
			Category: data.Category, Author: data.Author})
	}
	fmt.Println(pbbooks)
	return &pb.GetAllResponse{Book: pbbooks}, nil
}

func main() {
	// log if go crash, with the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// get env vars
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// mongoLocal := os.Getenv("MONGO_LOCAL")
	mongoImage := os.Getenv("MONGO_IMAGE")

	// create the mongo context
	mongoCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// connect MongoDB
	fmt.Println("Connecting to MongoDB...")
	client, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(mongoImage))
	if err != nil {
		log.Fatalf("Error Starting MongoDB Client: %v", err)
	}

	// check the connection
	err = client.Ping(mongoCtx, nil)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	} else {
		fmt.Println("Connected to Mongodb")
	}

	bookDB = client.Database("Bookstore").Collection("books")

	fmt.Println("Starting Listener...")
	l, err := net.Listen("tcp", "0.0.0.0:9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	pb.RegisterBookstoreServer(s, &server{})

	// Start a GO Routine
	go func() {
		fmt.Println("Bookstore Server Started...")
		if err := s.Serve(l); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait to exit (Ctrl+C)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block the channel until the signal is received
	<-ch
	fmt.Println("Stopping Bookstore Server...")
	s.Stop()
	fmt.Println("Closing Listener...")
	l.Close()
	fmt.Println("Closing MongoDB...")
	client.Disconnect(mongoCtx)
	fmt.Println("All done!")
}
