package bloomFilter
//Implements a fast and relatively efficient bloom filter.
//	It is fairly customizable in terms of attributes. 2 usable levels of buckets, and a completely customizable
//	amount of hash function iterations.
//
//At the max bucket level it will consume up to 0.5GB of RAM.

import(

	"fmt" //for yelling at people who use obscene data depth!
	"crypto/sha256" //for hashing... you know.... an important part of bloom filters
	"math/big" //for dealing with integer exponentiation. Yes, it is harder than it sounds in go
	"bytes" //for getting indices
	"encoding/binary" //similarly, for converting bytes to ints for indices!

	"math" //for estimating the accuracy of the given filter and setting buckets

	"encoding/json" //for serialization
	
	//the following three packages are used to compress the serialized data
	"compress/gzip"
	"io/ioutil"
	"io"

	//for benchmarking, used by external programs
	"crypto/rand"
	"time"
)

//takes the data to be hashed and a count of times to hash said data.
//uses sha256 for speed.
func hash(data []byte, iterations int) [][]byte {
	aHasher:= sha256.New()

	var hashes [][]byte
	var aHash []byte

	for i:= 0;i<iterations;i++{

		//get the hash
		aHasher.Write(data)
		aHash = aHasher.Sum( nil )

		//work with the hash
		hashes = append(hashes, aHash)
		data = aHash

		//clean the hasher before the next iteration!
		aHasher.Reset()
	}

	//return the set of hashes!
	return hashes
}

//takes a base and the exponent.
	//does dirty things with the big math package that make me sad. This is a necessary evil.
func intExponent(base, exponent int) int {
	valueBig:= new( big.Int ).Exp( big.NewInt(int64(base)), big.NewInt( int64( exponent)), nil )
	value:= int(valueBig.Int64())

	return value
}

//assumes that all input is <=8 bytes
//
//will append zeroes if required. Silently fails if given a greater than
//64 bits
func bytesToInt( someBytes []byte ) int {
	var value uint64



	//make sure the buffer is the proper size for an int64
	//therefore, 8 bytes!
	for len(someBytes)<8{
		someBytes = append(someBytes, byte(0) )
	}


	buf:= bytes.NewReader( someBytes )
	err:= binary.Read( buf, binary.LittleEndian, &value )
	if err!=nil{
		fmt.Println("Failure to convert bytes to int, that's bad.")
		panic(err)
	}

	return int(value)
}

//define a bloom filter using sha256, this is a basic bloom array.
// Always call yourBloomFilter.BuildBuckets before doing anything else or you will get yelled at. And it will be awkward.
//		!!only ever add or check. no delete is present and not desired
//	iterations of sha256 upon the same data provides a random distribution while
//	using only a single hash function that is known to be fast and relatively collision resistant
type BloomFilter struct{
	//how many times to run the filter's function. 
		//this is the k in terms of calculating accuracy
	HashIterations int

	//defines the bytes to use from each hash. at a single byte, there are only 256 possible buckets
		//this scales rapidly and, as the objects we use in go are at least a byte, 
		//will result in exponential memory usage growth
			//but that's fine because all the memory is allocated at the time when
			//we initialize the filter
	DataDepth int

	//we keep the actual data here, by using arrays of int64s and bitwise
	// operations, we can cut the memory used vs a straight array of bools
	// to 1/8. this is the difference between half a gig of usage vs 4 gig!
	IntBuckets []uint64

}

//builds the buckets for bloom filter.
	//essentially a reset switch
func (aBloomFilter *BloomFilter) BuildBuckets() {
	//make sure DataDepth is never, ever, ever,ever,ever,ever,ever above 4.
	//that means it'll attempt to use 2^(5*8) bytes which is big. REALLY DAMN BIG
	if aBloomFilter.DataDepth > 4{
		fmt.Println("you went above 4 for DataDepth.\n That means you just tried to create a 1099511627776 byte array. \n YOU DO NOT HAVE THAT MUCH MEMORY TO WORK WITH!")
		panic("you know what you have tried to do! \n This is what you deserve!")
	}

	//determine the total amount of buckets to build
		//this is defined by 2 to the power of the DataDepth * 8

	//for the love of god, don't look at that function, it will cause sufferring.
	possibleBuckets:= intExponent( 2, aBloomFilter.DataDepth*8 )

	//set up the int bucket
		//it is initialized to a 0 value at each integer.
		//since there are 64 usable bits per integer, we can
		//divide the possible number of buckets by 64
			//as a 64 bit is 8 bytes wide, we use an eighth of memory relative
			//to the naive route of a simple bool array where each bool is a byte!
	aBloomFilter.IntBuckets = make( []uint64, possibleBuckets / 64 )

}

//literally BuildBuckets in that it wipes the filter
//while maintaining its constants!
//Why does this exist? For convention mostly
func (aBloomFilter *BloomFilter) Reset() {
	aBloomFilter.BuildBuckets()
}

//sets the given bucket to filled
func (aBloomFilter *BloomFilter) Set(index int) {

	//using bitwise operations, get the integer in the array to use
		//go will just perform an division which rounds down into a integer,
		//perfect for our purposes
	integerToUse:= uint(index / 64) //as we are using 64 bits per integer

	//now that we have the integer to use, we need the bit to use
	bitToUse:= uint8( math.Mod( float64(index), 64 ))

	// set the bit using the following scheme where x is the int modified and position is a unsigned int.
	//	x = x | 1<<position
	aBloomFilter.IntBuckets[integerToUse] = aBloomFilter.IntBuckets[integerToUse] | 1<< bitToUse
}

//returns whether the given bucket is filled or not
func (aBloomFilter *BloomFilter) Get(index int) bool {

	//using bitwise operations, get the integer in the array to use
	//go will just perform an division which rounds down into a integer,
	//perfect for our purposes
	integerToUse:= index / 64 //as we are using 64 bits per integer

	//now that we have the integer to use, we need the bit to use
	bitToUse:= math.Mod( float64(index), 64 )

	//find if the bit is set to one by determining if the bit being set to true
	//results in the same integer using the following scheme
	//	bit := x & (1 << 63)

	bit:= (aBloomFilter.IntBuckets[integerToUse] & (1<< uint8( bitToUse))) >> uint8(bitToUse)

	if bit== 0{
		return false
	}
	
	return true

}

//gets the indices of the given data in terms of the constants the
//specific filter has
//
//not exported for good reason!
func (aBloomFilter *BloomFilter) getIndices( data []byte) []int {

	var hashes [][]byte

	//Get the hashes
	hashes = hash( data, aBloomFilter.HashIterations )

	//Convert each truncated hash(of size DataDepth in bytes) to an index into the bucket array
		//allocate all the working arrays now to prevent reallocation later.
	indices := make( []int, aBloomFilter.HashIterations)
	workingBytes := make( []byte, aBloomFilter.DataDepth )
	for i,aHash:= range hashes{

		//truncate the hash to the first BloomFilter.DataDepth bytes
		workingBytes = aHash[0:aBloomFilter.DataDepth]

		//get the index and append it to the indices. 
		indices[i] = bytesToInt(workingBytes)

	}

	return indices
}

//takes an array of bytes and adds it to the given bloom filter.
//very simple to use when the bloom array was set up properly
func (aBloomFilter *BloomFilter) Add( data []byte ) {

	//get the indices for the filter's buckets!
	indices:= aBloomFilter.getIndices(data)

	for _,anIndex:= range indices{
		aBloomFilter.Set(anIndex)
	}

}

//takes an array of bytes and checks its membership in the filter.
//returns a bool of membership
func (aBloomFilter *BloomFilter) CheckMembership( data []byte) bool {

	//get the indices for the filter's buckets
	indices:= aBloomFilter.getIndices(data)

	//check if each index is true.
	// the moment we hit a negative then the membership fails
	for _, anIndex:=range indices{
		if !aBloomFilter.Get(anIndex){
			return false
		}
	}

	//if we got this far, then the item is in the filter.
	return true


}

//serializes a bloom filter into a retrievable format for later usage
//
//takes the given name to use for the file and if to compress the file using gzip
func (aBloomFilter *BloomFilter) Serialize(fileName string, compress bool) error {
	//uses json for portability

	marshaled, err:= json.Marshal(aBloomFilter)
	if err!=nil{
		return err
	}


	if compress==false{
		//just save the data if compression is unneeded
		err:= ioutil.WriteFile( fileName, marshaled, 0664 )
		if err!=nil{
			return err
		}
		return nil
	}else{
		//compress the data if needed
		var b bytes.Buffer
		w:= gzip.NewWriter( &b )
		w.Write( marshaled )
		w.Close()
		err:= ioutil.WriteFile( fileName, b.Bytes(), 0664 )
		if err!=nil{
			return err
		}
	}

	return nil

}

//attempts to deserialize a file into a bloom filter.
//the counterpart to the above Serialize.
func RetrieveFilter(fileName string, compressed bool) (BloomFilter, error) {

	var aBloomFilter BloomFilter

	//retrieve the file
	data, err:= ioutil.ReadFile(fileName)
	if err!=nil{
		return aBloomFilter, err
	}

	var workingData []byte

	//if compressed, decompress before handing it off to the json umarshaller
	if compressed==false{
		workingData = data
	}else{
		var actualData bytes.Buffer

		var b bytes.Buffer
		b.Write( data )
		reader, err:= gzip.NewReader(&b)
		if err!=nil{
			return aBloomFilter,err
		}
		io.Copy(&actualData, reader)
		reader.Close()

		workingData = actualData.Bytes()


	}


	err= json.Unmarshal(workingData, &aBloomFilter)
	if err!=nil{
		return aBloomFilter, err
	}

	return aBloomFilter, nil
}

// gets an array of random bytes from the crypto generator
func getArrayOfRandBytes(arrayLength int) []byte {
	workingArray:= make([]byte, arrayLength)

	io.ReadFull(rand.Reader, workingArray)

	return workingArray
}

//returns a random integer between 0 and 255.
func getRandByteInt() int {
	return int(getArrayOfRandBytes(1)[0])
}

//checks the speed of the add function for given generic filter
func Bench(iterations, DataDepth int) uint {

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: iterations, DataDepth:DataDepth}
	workingFilter.BuildBuckets()

	iterationsToRun:= 100000

	//start the timer!
	start:= time.Now()
	for i := 0; i < iterationsToRun; i++ {
		workingFilter.Add( getArrayOfRandBytes(getRandByteInt()) )
	}

	elapsed:= time.Since(start)

	addsPerSecond:= float64( iterationsToRun) / (elapsed.Seconds() )


	return uint(addsPerSecond)
}