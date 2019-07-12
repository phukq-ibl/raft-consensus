raft:
	go build -o ./build/main ./cmd/raft/main.go 

run:
	go run  ./cmd/raft/main.go
run1: 
	./build/main -port 3000 -peers localhost:3000,localhost:3001,localhost:3002,localhost:3003,localhost:3004 -id 1
run2:
	./build/main -port 3001 -peers localhost:3000,localhost:3001,localhost:3002,localhost:3003,localhost:3004 -id 2
run3:
	./build/main -port 3002 -peers localhost:3000,localhost:3001,localhost:3002,localhost:3003,localhost:3004 -id 3
run4:
	./build/main -port 3003 -peers localhost:3000,localhost:3001,localhost:3002,localhost:3003,localhost:3004 -id 4
run5:
	./build/main -port 3004 -peers localhost:3000,localhost:3001,localhost:3002,localhost:3003,localhost:3004 -id 5