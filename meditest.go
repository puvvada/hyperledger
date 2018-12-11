package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
}

type Patient struct {
	MPI   string `json:"MPI"`
	FName string `json:"FName"`
	LName string `json:"LName"`
	Files string `json:"Files"`
	/* Files []struct {
		FileName    string `json:"FileName"`
		FileURL     string `json:"FileUrl"`
		CreatedDate string `json:"CreatedDate"`
	} `json:"Files"` */
	CreatedDate string `json:"CreatedDate"`
}

// Init is called during chaincode instantiation to initialize any
// data. Note that chaincode upgrade also calls this function to reset
// or to migrate data.
func (t *SimpleAsset) Init(stub shim.ChaincodeStubInterface) peer.Response {
	// Get the args from the transaction proposal
	args := stub.GetStringArgs()
	if len(args) != 2 {
		return shim.Error("Incorrect arguments. Expecting a key and a value")
	}

	// Set up any variables or assets here by calling stub.PutState()

	// We store the key and the value on the ledger
	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create asset: %s", args[0]))
	}
	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode. Each transaction is
// either a 'get' or a 'set' on the asset created by Init function. The Set
// method may create a new asset by specifying a new key-value pair.
func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	// Extract the function and args from the transaction proposal
	fn, args := stub.GetFunctionAndParameters()

	var result string
	var err error
	if fn == "init_patient" {
		result, err = init_patient(stub, args) //save the patient by MPI , if MPi is present then update the Patient with new block
	} else if fn == "get_patient" { // get Patient by MPi
		result, err = get_patient(stub, args)
	} else if fn == "get_TxHisBypatId" { //get patient transactions by Id
		return getHistoryByPatientId(stub, args)
	} else if fn == "get_AllPatients" {
		return read_everything(stub) //get all patients in the blockchain
	}

	if err != nil {
		return shim.Error(err.Error())
	}

	// Return the result as success payload
	return shim.Success([]byte(result))
}

// Set stores the asset (both key and value) on the ledger. If the key exists,
// it will override the value with the new one
func init_patient(stub shim.ChaincodeStubInterface, args []string) (string, error) {

	if len(args) != 5 {
		return "", fmt.Errorf("Incorrect arguments. Expecting proper input")
	}
	fmt.Println("========Begin of Pat Reg======")
	pat := Patient{MPI: args[0], FName: args[1], LName: args[2], Files: args[3], CreatedDate: time.Now().UTC().String()}
	UserBytes, _ := json.Marshal(pat)
	err := stub.PutState(pat.MPI, UserBytes)
	if err != nil {
		return "", fmt.Errorf("Failed to set asset: %s", UserBytes)
	}
	fmt.Println("========End of Pat Reg======")
	return pat.MPI, nil
}

// Get returns the value of the specified asset key
func get_patient(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key")
	}
	fmt.Println("========Begin of Pat get by MPi======")
	value, err := stub.GetState(args[0])
	if err != nil {
		return "", fmt.Errorf("Failed to get asset: %s with error: %s", args[0], err)
	}
	if value == nil {
		return "", fmt.Errorf("Asset not found: %s", args[0])
	}
	fmt.Println("========End of Pat get by MPi======")
	return string(value), nil
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}

//function for reading everything from the blockchain
func read_everything(stub shim.ChaincodeStubInterface) peer.Response {
	type Everything struct {
		Patients []Patient `json:"patients"`
	}
	var everything Everything
	fmt.Println("Started pulling all the patients data")
	//Get all the patients
	resultsIterator, err := stub.GetStateByRange("MPI01", "MPI99999999999")
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()
	fmt.Println("iterating pulled patients data")
	for resultsIterator.HasNext() {
		aKeyValue, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		queryKeyAsStr := aKeyValue.Key
		queryValAsBytes := aKeyValue.Value
		fmt.Println("On Patientid - ", queryKeyAsStr)
		var patient Patient
		json.Unmarshal(queryValAsBytes, &patient)                  //unstringify it ie json.parse()
		everything.Patients = append(everything.Patients, patient) //adding patient to list
	}
	fmt.Println("completed iterating pulled patients data")
	fmt.Println("Patient Array", everything.Patients)

	//change to array of bytes
	everythingAsBytes, _ := json.Marshal(everything) //converting to  array of bytes

	fmt.Println("Completed pulling all the patients data")
	return shim.Success(everythingAsBytes)
}

//get history of patient completely by key
func getHistoryByPatientId(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	type AuditHistory struct {
		TxId  string  `json:"txId"`
		Value Patient `json:"value"`
	}

	var history []AuditHistory
	var patient Patient

	if len(args) != 1 {
		return shim.Error("Incorrect Number of Arguments,Expecting 1-PatientId")
	}

	patienId := args[0]
	fmt.Println("-Start getting history for PatientId - %s\t", patienId)
	//get history
	resultsIterator, err := stub.GetHistoryForKey(patienId)
	if err != nil {
		fmt.Println("-Unable to get for PatientId - %s\t", patienId, err.Error)
		return shim.Error(err.Error())
	}

	defer resultsIterator.Close()
	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		var tx AuditHistory
		tx.TxId = historyData.TxId
		json.Unmarshal(historyData.Value, &patient)
		if historyData.Value == nil {
			var emptyPatient Patient
			tx.Value = emptyPatient
		} else {
			json.Unmarshal(historyData.Value, &patient)
			tx.Value = patient
		}
		history = append(history, tx)
	}
	fmt.Printf("- get History for Patient returning:\n%s", history)
	//change the array of bytes
	historyAsBytes, _ := json.Marshal(history)
	return shim.Success(historyAsBytes)
}
