package aursir4go

import (
	"errors"
	"time"
)
//An ImportedAppKey represents an applications imports and is used to call and listen to functions and to create
// callchains
type ImportedAppKey struct {
	iface      *AurSirInterface
	key        AppKey
	tags       []string
	importId   string
	Connected  bool
	listenFuns []string
	listenChan chan Result
	persistenceStrategies map[string] string
}

//Tags returns the currently registered tags for the import.
func (iak *ImportedAppKey) Tags() []string {
	return iak.tags
}

//Name returns the name of the imports ApplicationKey.
func (iak *ImportedAppKey) Name() string {
	return iak.key.ApplicationKeyName
}

//Listen to functions registers the import for listening to this function. Use Listen to get Results for this function.
func (iak *ImportedAppKey) ListenToFunction(FunctionName string) {
	listenid := iak.key.ApplicationKeyName + "." + FunctionName
	iak.iface.registerResultChan(listenid, iak.listenChan)
	iak.iface.out <- AurSirListenMessage{iak.importId, FunctionName}
	iak.listenFuns = append(iak.listenFuns, listenid)
}

//Listen listens for results on listened functions. If no listen functions have been added, it returns an empty result Result.
func (iak *ImportedAppKey) Listen() Result {
	if len(iak.listenFuns) == 0 {
		return Result{}
	}
	return <-iak.listenChan
}

//Call functions calls the function specified by FunctionName and returns a channel to get the result. This channel we
// be nil on Many2... call types! You need to use Listen() in this case.
func (iak *ImportedAppKey) CallFunction(FunctionName string, Arguments interface{}, CallType int64) (chan Result, error) {
	return iak.callFunction(FunctionName,Arguments,CallType, false)
}


//UpdateTags sets the imports tags while overriding the old and registers the new tagset at the runtime. If you want to
// add a tag, use AddTag.
func (iak *ImportedAppKey) UpdateTags(NewTags []string) {
	iak.tags = NewTags
	iak.iface.out <- AurSirUpdateImportMessage{iak.importId, iak.tags}
}

//AddTag adds a tag to the imports tags and registers the new tagset at the runtime. If you want to set a new tagset,
// use UpdateTags
func (iak *ImportedAppKey) AddTag(Tag string) {
	iak.UpdateTags(append(iak.tags,Tag))
}

func (iak *ImportedAppKey) NewCallChain(OriginFunctionName string, Arguments interface{}, OriginCallType int64) (CallChain, error) {
	if OriginCallType > 3 {
		return CallChain{}, errors.New("Invalid calltype")
	}

	codec := GetCodec("JSON")
	args, err := codec.Encode(Arguments)
	if err != nil {
		return CallChain{}, err
	}

	cc := createCallChain(iak.iface)
	cc.setOrigin(iak.key.ApplicationKeyName, OriginFunctionName, "JSON", &args, iak.Tags(), OriginCallType, iak.importId)
	return cc, nil

}

func (iak *ImportedAppKey) FinalizeCallChain(FunctionName string, ArgumentMap map[string]string, CallType int64, CallChain CallChain) (chan Result, error) {
	if CallType > 3 {
		return nil, errors.New("Invalid calltype")
	}

	reqUuid := generateUuid()
	fcc := ChainCall{
		iak.key.ApplicationKeyName,
		FunctionName,
		ArgumentMap,
		CallType,
		iak.Tags(),
		reqUuid}
	CallChain.finalImportId = iak.importId
	CallChain.finalCall = fcc
	err := CallChain.Finalize()

	if err != nil {
		return nil, err
	}

	var resChan chan Result
	if CallType == ONE2ONE || CallType == ONE2MANY {
		resChan = make(chan Result)
		iak.iface.registerResultChan(reqUuid, resChan)
	}

	return resChan, nil
}

//SetLogging sets the persitence strategy for function calls of the specified function to "log". This overrides all previous
// persitence strategies!
func (iak *ImportedAppKey) SetLogging(FunctionName string){
	iak.persistenceStrategies[FunctionName] = "log"
}

func (iak *ImportedAppKey) PersistentCallFunction(FunctionName string, Arguments interface{}, CallType int64) (chan Result, error) {
	return iak.callFunction(FunctionName,Arguments,CallType, true)
}
func (iak *ImportedAppKey) callFunction(FunctionName string, Arguments interface{}, CallType int64, Persist bool) (chan Result, error) {

	if CallType > 3 {
		return nil, errors.New("Invalid calltype")
	}

	codec := GetCodec("JSON")
	if codec == nil {
		return nil, errors.New("unknown codec")
	}

	args, err := codec.Encode(Arguments)
	if err != nil {
		return nil, err
	}

	reqUuid := generateUuid()

	iak.iface.out <- AurSirRequest{
		iak.key.ApplicationKeyName,
		FunctionName,
		CallType,
		iak.tags,
		reqUuid,
		iak.importId,
		time.Now(),
		"JSON",
		false,
		Persist,
		"",
		args,
		false,
		false,
	}

	var resChan chan Result
	if CallType == ONE2ONE || CallType == ONE2MANY {
		resChan = make(chan Result)
		iak.iface.registerResultChan(reqUuid, resChan)
	}

	return resChan, nil
}
