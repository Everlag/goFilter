package bloomFilter

import (

	"testing"
	"fmt"
	"crypto/sha256"
	"runtime"

)

//last known to perfectly work form of hash()
func oldHash(data []byte, iterations int) [][]byte {
	aHasher:= sha256.New()

	var hashes [][]byte
	var aHash []byte

	readyData:= data

	for i:= 0;i<iterations;i++{

		//get the hash
		aHasher.Write(readyData)
		aHash = aHasher.Sum( nil )

		//work with the hash
		hashes = append(hashes, aHash)
		readyData = aHash

		//clean the hasher before the next iteration!
		aHasher.Reset()
	}

	//return the set of hashes!
	return hashes
}

//these define the expected behaviour for the filter
//when boolean buckets are utilized
//
//this is essentially a copy - paste from the naive implementation that
//existed from before the int based direct bit access method
//
//this is known to work
//
//comments have been stripped for ease of use
type boolBloomFilter struct{

	HashIterations int

	DataDepth int


	Buckets []bool


	Ready bool


	GCSeparation int

	currentGCSeparation int
}

func (aBloomFilter *boolBloomFilter) BuildBuckets() {
	if aBloomFilter.DataDepth > 4{
		fmt.Println("oh jesus, you went above 4 for DataDepth.\n That means you just tried to create a 1099511627776 byte array. \n YOU DO NOT HAVE THAT MUCH MEMORY TO WORK WITH!")
		panic("you know damn well what you have tried to do! \n This is what you deserve!")
	}
	possibleBuckets:= intExponent( 2, aBloomFilter.DataDepth*8 )

	aBloomFilter.Buckets = make([]bool, possibleBuckets)

	aBloomFilter.Ready = true
}

func (aBloomFilter *boolBloomFilter) Reset() {
	aBloomFilter.BuildBuckets()
}

func (aBloomFilter *boolBloomFilter) Set(index int) {
	aBloomFilter.Buckets[index] = true

}

func (aBloomFilter *boolBloomFilter) Get(index int) bool {
	return aBloomFilter.Buckets[index]

}

func (aBloomFilter *boolBloomFilter) getIndices( data []byte) []int {
	aBloomFilter.currentGCSeparation++

	if aBloomFilter.currentGCSeparation>=aBloomFilter.GCSeparation{
		runtime.GC()
	}

	var hashes [][]byte

	hashes = oldHash( data, aBloomFilter.HashIterations )

	indices := make( []int, aBloomFilter.HashIterations)
	workingBytes := make( []byte, aBloomFilter.DataDepth )
	for i,aHash:= range hashes{
		workingBytes = aHash[0:aBloomFilter.DataDepth]
		indices[i] = bytesToInt(workingBytes)

	}

	return indices
}

func (aBloomFilter *boolBloomFilter) Add( data []byte ) {
	if aBloomFilter.Ready!= true{
		panic("You didn't read what the Ready bool was used for, did you?\n You deserve this.")
	}

	indices:= aBloomFilter.getIndices(data)

	for _,anIndex:= range indices{
		aBloomFilter.Set(anIndex)
	}

}

func (aBloomFilter *boolBloomFilter) CheckMembership( data []byte) bool {
	if aBloomFilter.Ready!=true{
		panic("You didn't read what the Ready bool was used for, did you?\n You deserve this.")
	}

	indices:= aBloomFilter.getIndices(data)

	for _, anIndex:=range indices{
		if aBloomFilter.Get(anIndex)!=true{
			return false
		}
	}

	return true


}


//compares the reduced memory footprint integer method with the
//naive boolean bucket method
func TestFilterValidity(t *testing.T) {
	//initialize the filter
	workingFilter:= BloomFilter{HashIterations: standardHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//initialize the boolean buckets
	naiveFilter:= boolBloomFilter{HashIterations: standardHash, DataDepth:4, GCSeparation:5}
	naiveFilter.BuildBuckets()

	//defines the quantity of 8 byte values to use for the test
	testingLength:= 200
	testBytes:= make([][]byte, testingLength)

	//fill the testing byte array with random cryptographically secure bytes.
	for i := 0; i < testingLength; i++ {
		testBytes[i] = getArrayOfRandBytes(8)
	
		//might as well fill the bloom filter in the same loop
		workingFilter.Add( testBytes[i] )

		//and fill the boolean buckets
		naiveFilter.Add( testBytes[i] )
	}

	var randomFail bool
	var failures uint32

	//now check for values that are highly unlikely to be in the filter:
		//this is a false positive test
	for i := 0; i < testingLength; i++ {
		randomData:= getArrayOfRandBytes(8)

		if workingFilter.CheckMembership(randomData)!= naiveFilter.CheckMembership(randomData){
			fmt.Println( workingFilter.CheckMembership(randomData) , naiveFilter.CheckMembership(randomData))
			fmt.Println(randomData)
			failures++
			randomFail = true
		}
	}

	if randomFail== true{
		t.Error("Filters failed to match during random data testing for the following quantity of iterations", failures)
	}

	var properFail bool

	for i := 0; i < testingLength; i++ {
		
		//test to make sure the values we added return properly

		if workingFilter.CheckMembership(testBytes[i])!= naiveFilter.CheckMembership(testBytes[i]){
			fmt.Println( workingFilter.CheckMembership(testBytes[i]) , naiveFilter.CheckMembership(testBytes[i]))
			fmt.Println(testBytes[i])
			failures++
			properFail = true
		}

	}

	if properFail== true{
		t.Error("Filters failed to match during set data testing for the following quantity of iterations", failures)
	}


}

func (aBloomFilter *BloomFilter) randomFill( iterations int ) {
	
	for i := 0; i < iterations; i++ {
		
		aBloomFilter.Add( getArrayOfRandBytes( int(getArrayOfRandBytes(1)[0]) ) )

	}

}

//makes sure the setting and getting functions of the bloom filter
//are operating as expected for many single bytes.
//
//single bytes are used to ensure there is at least one collisions of input values
//to make sure everything operates as usual under those circumstances
func TestFilterIO( t *testing.T ) {
	workingFilter:= BloomFilter{HashIterations: largeHash, DataDepth:3}
	workingFilter.BuildBuckets()

	//get the data to add
	dataToAdd:= getArrayOfRandBytes(125)

	//add it via the high level function which uses the low level .Set
		
	for i := range dataToAdd{
		
		workingFilter.Add( []byte{ dataToAdd[i]} )

	}

	var hadAnIssue bool

	//make sure it was added properly!
	for i := range dataToAdd{
		
		if workingFilter.CheckMembership( []byte{ dataToAdd[i]} ) == false{
			hadAnIssue = true
			fmt.Println( workingFilter.CheckMembership([]byte{ dataToAdd[i]}) )
			fmt.Println(dataToAdd[i])
		}

	}

	if hadAnIssue == true{
		t.Error("Failled to have filter output true for all data input")
	}

}

/*

func TestSerialize(t *testing.T) {
		//initialize the filter
	workingFilter:= BloomFilter{HashIterations: standardHash, DataDepth:3}
	workingFilter.BuildBuckets()


	//make the filter interesting
		//defines the quantity of 8 byte values to use for the test
	testingLength:= 300
	testBytes:= make([][]byte, testingLength)

	//fill the testing byte array with random cryptographically secure bytes.
	for i := 0; i < testingLength; i++ {
		testBytes[i] = getArrayOfRandBytes(8)
		workingFilter.Add( testBytes[i] )
	}

	//serialize it and send it to the file system
	err:=workingFilter.Serialize("testingFilter.json",false)
	errComp:=workingFilter.Serialize("testingFilterCompressed.json",true)
	if err!=nil || errComp!=nil{
		t.Error("Failed to serialize the filter!")
	}

	//deserialize it and match it to the filter it was serialized from.
	basicFilter, err:= RetrieveFilter("testingFilter.json", false)
	compressedFilter, errComp:= RetrieveFilter("testingFilterCompressed.json", true)
	if err!=nil || errComp!=nil{
		t.Error("Failed to deserialize the filters!")
	}

	var regularFail bool

	for i := 0; i < testingLength; i++ {
		
		//test to make sure the values we added return properly

		if workingFilter.CheckMembership(testBytes[i])!= basicFilter.CheckMembership(testBytes[i]){
			fmt.Println( workingFilter.CheckMembership(testBytes[i]) , basicFilter.CheckMembership(testBytes[i]))
			fmt.Println(testBytes[i])
			regularFail = true
		}

	}

	if regularFail{
		t.Error("Basic filter when retrieved is not equal to original filter")
	}

	var compressedFail bool

	for i := 0; i < testingLength; i++ {
		
		//test to make sure the values we added return properly

		if workingFilter.CheckMembership(testBytes[i])!= compressedFilter.CheckMembership(testBytes[i]){
			fmt.Println( workingFilter.CheckMembership(testBytes[i]) , compressedFilter.CheckMembership(testBytes[i]))
			fmt.Println(testBytes[i])
			compressedFail = true
		}

	}


	if compressedFail{
		t.Error("compressedFilter filter when retrieved is not equal to original filter")
	}
}
*/

const(
	largeHash = int( 1e4)
	moderateHash = int( 1e3)
	standardHash = int( 1e1)
)

//checks the speed of the add function for the generated filter
func BenchmarkAddSpeedLargeHash(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: largeHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .Add benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.Add( getArrayOfRandBytes(getRandByteInt()) )
	}

}

//checks the speed of the add function for the generated filter
func BenchmarkAddSpeedModerateHash(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: moderateHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .Add benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.Add( getArrayOfRandBytes(getRandByteInt()) )
	}

}

//checks the speed of the add function for the generated filter
func BenchmarkAddSpeedStandardHash(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: standardHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .Add benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.Add( getArrayOfRandBytes(getRandByteInt()) )
	}

}

//checks the speed of the add function for the last known
//to work perfectly due to naivity filter
func BenchmarkAddSpeedStandardHashRef(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= boolBloomFilter{HashIterations: standardHash, DataDepth:4, GCSeparation:1e10}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .Add benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.Add( getArrayOfRandBytes(getRandByteInt()) )
	}

}


//checks the speed of the checkMembership function for the generated filter
func BenchmarkCheckSpeedLargeHash(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: largeHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .checkMembership benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.CheckMembership( getArrayOfRandBytes(getRandByteInt()) )
	}

}

//checks the speed of the checkMembership function for the generated filter
func BenchmarkCheckSpeedModerateHash(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: moderateHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .checkMembership benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.CheckMembership( getArrayOfRandBytes(getRandByteInt()) )
	}

}

//checks the speed of the checkMembership function for the generated filter
func BenchmarkCheckSpeedStandardHash(b *testing.B) {
	//fmt.Println("Setting up the filter's benchmark environment")

	//perform expensive setup
	workingFilter:= BloomFilter{HashIterations: standardHash, DataDepth:4}
	workingFilter.BuildBuckets()

	//fmt.Println("Starting .checkMembership benchmark!")

	b.ResetTimer()

	//start the timer!
	for i := 0; i < b.N; i++ {
		workingFilter.CheckMembership( getArrayOfRandBytes(getRandByteInt()) )
	}

}