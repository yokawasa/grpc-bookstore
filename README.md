# grpc-bookstore
Simple Bookstore CRUD service using gRPC and mongoDB

## Quickstart

Start mongoDB server using docker-compose

```bash
docker compose up -d
```

Check if the server is running by `docker-compose ps`

```bash
docker compose ps

NAME                         IMAGE               COMMAND                  SERVICE             CREATED             STATUS              PORTS
grpc-bookstore-db-client-1   mongo-express       "tini -- /docker-ent…"   db-client           5 minutes ago       Up 11 seconds       0.0.0.0:8081->8081/tcp
mongo-server                 mongo               "docker-entrypoint.s…"   db                  5 minutes ago       Up 12 seconds       0.0.0.0:27017->27017/tcp
```

Setup `.env` file where MONGO_IMAGE is defined like this:

```bash
cat << EOF | tee .env
MONGO_IMAGE="mongodb://mongoadmin:secret@localhost:27017/"
EOF
```
note: you can simply copy .env.example on the same directory

Finally run gRPC server

```bash
 go run main.go 
```

Expected output would be like this

```
Connecting to MongoDB...
Connected to Mongodb
Starting Listener...
Bookstore Server Started...
```

After the test, shutdown the mongo server

```bash
docker compose down
```

## Testing the bookstore server

PostBook request for the endpoint `0.0.0.0:9090`:

```json
{
  "book": {
    "bookID": "12345",
    "bookName": "sample book1",
    "category": "novel",
    "author": "Haruki Murakami"
  }
}
```

Updatebook request:

```json
{
  "book": {
    "bookID": "12345",
    "bookName": "sample book2",
    "category": "novel",
    "author": "Haruki Murakami"
  }
}
```

DeleteBook request:

```json
{
  "id": "<bookid>"
}
```

GetBook request:

```json
{
  "id": "<bookid>"
}
```

With GetAllBooks request, you'll get all books. Expected response would be like this:

```json
{
  "book": [
    {
      "bookID": "12345",
      "bookName": "sample book1",
      "category": "comic",
      "author": "haruki murakami"
    },
    {
      "bookID": "officia mollit voluptate dolore",
      "bookName": "officia occaecat fugiat esse",
      "category": "officia in dolor esse",
      "author": "non ut Excepteur"
    }
  ]
}
```
