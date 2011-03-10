package master

import (
	"net"
	"container/vector"
)

type server struct {
	addr net.TCPAddr
	capacity uint64
	chunks *vector.Vector
}

type heapCommand struct {
	command uint64
	server interface {}
}
type serverHeap struct {
	serverChan chan * heapCommand
	vec * vector.Vector
}

func (s * serverHeap) Len() int { 
	return s.vec.Len() 
}

func (s * serverHeap) Less(i, j int) bool { 
	si := s.vec.At(i).(*server)
	sj := s.vec.At(j).(*server)
	return (si.capacity/uint64(si.chunks.Len())) < (sj.capacity/uint64(sj.chunks.Len()))
}
func (s * serverHeap) Swap(i, j int)      {
	 s.vec.Swap(i,j)
}
func (s * serverHeap) Push(serv interface {}) {
	s.serverChan <- &heapCommand{0,serv}
	//s.vec.Push(serv)
}
func (s * serverHeap) Pop() interface {} {
	s.serverChan <- &heapCommand{1,nil}
	//command := new(heapCommand)
	command := <- s.serverChan
	return command.server
}
func (s * serverHeap) Handler() {

	for rec := range s.serverChan {
	
		if(rec.command == 0){
			s.vec.Push(rec.server)
		}
		if(rec.command == 1){
			s.serverChan <- &heapCommand{1,s.vec.Pop()}
		}
	}
}
