//this is the location for the client library
package client

import (
	"container/vector"
	"rpc"
	"os"
	"log"
	"io/ioutil"
	"../include/sfs"
	"net"
)

//NEED TO MAKE A TYPE FOR FILE
/*
	int fd
	list of chunks (preferably in order)
	list of serverlocations, one for each chunk 
	string filename
	int size

*/
const(
 O_CREATE = 1
 O_RDONLY
 O_WRONLY
 SEEK_SET
 SEEK_CURR
 SEEK_END
)


type file struct {
	size uint64
	filePtr  uint64
	chunkInfo *vector.Vector
}

var fd = 0
var openFiles = map[int] file{}


func Open( filename string , write bool, master string,  size uint64  ) (int){

	client,err :=rpc.Dial("tcp", master + ":1338"); //IP needs to be changed to Master's IP
	if err != nil{
		log.Printf("Client: Error", err.String());
		os.Exit(1)
		return -1
	}else{
	fileInfo := new (sfs.OpenReturn)
	fileArgs := new (sfs.OpenArgs)
	fileArgs.Name = filename
	fileArgs.Size = size
	client.Call("Master.ReadOpen", &fileArgs,&fileInfo)
	if fileInfo.New {
		log.Printf("\nClient: New file!\n")
	}
	fd++
	var nextFile file
	nextFile.size = fileInfo.Size
	nextFile.filePtr = 0
	nextFile.chunkInfo = new(vector.Vector)
	for i := 0 ; i < fileInfo.Chunk.Len(); i ++ {
		nextFile.chunkInfo.Push(fileInfo.Chunk)
	}

	openFiles[fd] = nextFile
	return fd;
	}
	return -1
}

/* read */
func Read (fd int, size int) (vector.Vector, int ){
	//goes to chunk and gets a chunk of memory to read...
	fileInfo := new (sfs.ReadReturn);
	fileArgs := new (sfs.ReadArgs);
	fdFile, inMap := openFiles[fd]
	var entireRead vector.Vector
	if !inMap {
		log.Printf("Client: File not in open list!\n")
		return entireRead, -1
	}
	index := 0;
	for i := 0; i<fdFile.chunkInfo.Len(); i++ {
		client,err :=rpc.Dial("tcp",fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).Servers.At(0).(*net.TCPAddr).String())
		if err != nil{
			log.Printf("Client: Dial Failed in Read")
			return entireRead, -1
		}
		fileArgs.ChunkIDs= fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).ChunkID;
		fileArgs.Offsets = 0;
		fileArgs.Lengths  = sfs.CHUNK_SIZE;

		chunkCall := client.Go("Server.Read", &fileArgs,&fileInfo, nil);
		replyCall:= <-chunkCall.Done
		if replyCall.Error!=nil{
			log.Printf("Client: error in reply from rpc in read\n");
			return entireRead, -1
		}
		log.Printf("\nClient: Status = %d\n",fileInfo.Status);
		log.Printf("Client: Data = %d\n",fileInfo.Data);
		if(fileInfo.Status!=0){
			break;
		}
		for j:=0; j<sfs.CHUNK_SIZE ; j++{
			entireRead.Push(fileInfo.Data.Data[j]);
			index++;
		}
	}

	return  entireRead, fileInfo.Status;
}

/* write */


func Write (fd int , data vector.Vector  ) (int){
	fileArgs := new (sfs.WriteArgs);
	fileInfo := new (sfs.WriteReturn);
	fdFile, inMap := openFiles[fd]
	if !inMap {
		log.Printf("Client: File not in open list!\n")
		return -1
	}
	index:=  0
	var size uint64
	var numChunks uint64
	size = uint64(data.Len());
	numChunks = size / sfs.CHUNK_SIZE
	if((size %sfs.CHUNK_SIZE)!= 0){
		numChunks++
	}
	for i := 0; i<int(numChunks); i++ {
		for j:=0; j<sfs.CHUNK_SIZE ; j++{
			if(index < int(size)){
				fileArgs.Data.Data[j] = data.At(j).(byte);
			}
			index++;
		}
		fileArgs.Info.ChunkID = (fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).ChunkID)
		fileArgs.Offset = 0;
		if((i != fdFile.chunkInfo.Len()-1)|| (size%sfs.CHUNK_SIZE==0)){
			fileArgs.Length = sfs.CHUNK_SIZE;
		}else{
			fileArgs.Length =uint(size)% uint(sfs.CHUNK_SIZE)
		}
		
		for {
			
			client,err :=rpc.Dial("tcp",fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).Servers.At(0).(*net.TCPAddr).String())
			if err != nil{
				log.Printf("Client: Dial Failed in write");
				if fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).Servers.At(1).(*net.TCPAddr) == nil {
					log.Fatal("Client: out of chunk servers to write to.")
				}
				fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).Servers.Slice(1,fdFile.chunkInfo.At(i).(*sfs.ChunkInfo).Servers.Len())
				continue
			}
			
			chunkCall := client.Go("Server.Write", &fileArgs,&fileInfo, nil);
			replyCall:= <-chunkCall.Done
			
			if replyCall.Error != nil{
				log.Fatal("Client: error in reply from rpc in write\n");
			}
			if(fileInfo.Status!=0){
				log.Fatal("Client: Status non zero =",fileInfo.Status)
			}
			break
		}
	}
	return fileInfo.Status;
}
/// no idea if this works at all
func WriteFromFile(fileNameLocal string, fileNameServer string, master string)(int){
	f, err := ioutil.ReadFile(fileNameLocal)
	if err!=nil {
	}
	fd := Open(fileNameServer, true, master, uint64(len(f)))

	toWrite := new(vector.Vector)
	for j:=0; j<len(f) ; j++{
		toWrite.Push(f[j])
	}
	//need to open a file and have a bunch of chunks
	return Write (fd , *toWrite  )
}

func ReadToFile(fileNameLocal string, fileNameServer string, master string)(int){
//	fd := Open(fileNameServer, true, master, uint64(len(f)))
// read , create local file, write to local file, close local file
	return 0;
}

/* delete */
//TODO
func Delete(filename string) (int){
	return -1;
}

/* close */
//TODO
func Close(fd int) (int){
	return -1;
}

func ReadDir(path string) (vector.Vector, int){
	var x vector.Vector
	return x,  -1;
}

/* seek */
//TODO
func Seek (fd int, offset int, whence int) (sfs.Chunk, int){

	fileInfo := new (sfs.ReadReturn);
//	fileArgs := new (sfs.ReadArgs);
//	fdFile, inMap := openFiles[fd]
//	if !inMap {
//		log.Printf("Client: File not in open list!\n")
//		return fileInfo.Data, -1
//	}
//	_,err :=rpc.Dial("tcp",fdFile.serverAddress.String())
//	if err != nil{
//		log.Printf("Client: Dial Failed")
//		return fileInfo.Data, -1
//	}
//	fileArgs.ChunkIDs= fdFile.chunk;
//	fileArgs.Offsets = 0;
//	fileArgs.Lengths = sfs.CHUNK_SIZE;
	//ChunkCall := client.Go("Server.Read", &fileArgs,&fileInfo, nil);
	//replyCall:= <-chunkCall.Done
	//this is asynchronous, probably want to change it to synchronous
	//if replyCall.Error!=nil{
	//	log.Printf("Client: error in reply from rpc in read\n");
	//	return fileInfo.Data, -1
	//}
	//log.Printf("\nClient: Status = %d\n",fileInfo.Status);
	//log.Printf("Client: Data = %d\n",fileInfo.Data);
	return  fileInfo.Data, fileInfo.Status;
}


