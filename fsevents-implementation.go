package main

// #cgo pkg-config: python-2.7
// #define Py_LIMITED_API
// #include <Python.h>
// int ParseOurArguments(PyObject *, PyObject **, PyObject **);
// PyObject* PyArg_BuildNone();
// PyObject* CallPythonFunction(PyObject*, char*, char*);
// void IncreaseReference(PyObject*);
// void DecreaseReference(PyObject*);
import "C"
import (
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsevents"
)

var es *fsevents.EventStream
var _callbacks map[string][]*C.PyObject

// PyStringAsString convert a python string to a Go string
func PyStringAsString(s *C.PyObject) string {
	return C.GoString(C.PyString_AsString(s))
}

// PyListOfStrings convert a list of objects to a list of strings
func PyListOfStrings(listObj *C.PyObject) []string {
	numLines := int(C.PyList_Size(listObj))

	if numLines < 0 {
		return nil
	}

	aList := []string{}
	for i := 0; i < numLines; i++ {
		strObj := C.PyList_GetItem(listObj, C.Py_ssize_t(i))
		aList = append(aList, PyStringAsString(strObj))
	}
	return aList
}

//export schedule
func schedule(self, args *C.PyObject) *C.PyObject {
	var argPaths *C.PyObject

	var callback *C.PyObject
	success := C.ParseOurArguments(args, &callback, &argPaths)

	if success == 0 {
		log.Fatal("Unable to parse the passed arguments", C.PyObject_Repr(args))
		C.PyErr_SetString(C.PyExc_TypeError, C.CString("invalid parameters"))
		return nil
	}

	if C.PyCallable_Check(callback) == 0 {
		log.Fatal("The first argument must be callable", C.PyObject_Repr(args))
		C.PyErr_SetString(C.PyExc_TypeError, C.CString("parameter must be callable"))
		return nil
	}
	C.IncreaseReference(callback)

	paths := PyListOfStrings(argPaths)
	if paths == nil {
		log.Fatal("Sorry, you should pass a list of paths as second argument.")
		C.PyErr_SetString(C.PyExc_TypeError, C.CString("use a list of paths as second argument"))
		return nil
	}

	if _callbacks == nil {
		_callbacks = make(map[string][]*C.PyObject)
	}

	_callbacks[paths[0]] = append(_callbacks[paths[0]], callback)
	dev, err := fsevents.DeviceForPath(paths[0])
	if err != nil {
		log.Fatalf("Failed to retrieve device for path: %v", err)
		C.PyErr_SetString(C.PyExc_TypeError, C.CString("failed to retrieve device for path"))
		return nil
	}

	es = &fsevents.EventStream{
		Paths:   paths,
		Latency: 500 * time.Millisecond,
		Device:  dev,
		Flags:   fsevents.FileEvents | fsevents.WatchRoot,
	}

	return C.PyArg_BuildNone()
}

var noteDescription = map[fsevents.EventFlags]string{
	fsevents.MustScanSubDirs: "MustScanSubdirs",
	fsevents.UserDropped:     "UserDropped",
	fsevents.KernelDropped:   "KernelDropped",
	fsevents.EventIDsWrapped: "EventIDsWrapped",
	fsevents.HistoryDone:     "HistoryDone",
	fsevents.RootChanged:     "RootChanged",
	fsevents.Mount:           "Mount",
	fsevents.Unmount:         "Unmount",

	fsevents.ItemCreated:       "Created",
	fsevents.ItemRemoved:       "Removed",
	fsevents.ItemInodeMetaMod:  "InodeMetaMod",
	fsevents.ItemRenamed:       "Renamed",
	fsevents.ItemModified:      "Modified",
	fsevents.ItemFinderInfoMod: "FinderInfoMod",
	fsevents.ItemChangeOwner:   "ChangeOwner",
	fsevents.ItemXattrMod:      "XAttrMod",
	fsevents.ItemIsFile:        "IsFile",
	fsevents.ItemIsDir:         "IsDir",
	fsevents.ItemIsSymlink:     "IsSymLink",
}

func callTheCallback(event fsevents.Event) {
	note := createNote(event)
	var callbacksToCall []*C.PyObject
	var minDistance = 200
	for k := range _callbacks {
		distance := len(strings.Replace(event.Path, k, "", 1))
		if distance < minDistance {
			minDistance = distance
			callbacksToCall = _callbacks[k]
		}
	}
	for _, callback := range callbacksToCall {
		result := C.CallPythonFunction(callback, C.CString(event.Path), C.CString(note))
		C.DecreaseReference(result)
	}
}

func createNote(event fsevents.Event) string {
	note := ""
	for bit, description := range noteDescription {
		if event.Flags&bit == bit {
			note += description + " "
		}
	}
	return note
}

func logEvent(event fsevents.Event) {
	note := createNote(event)
	log.Printf("EventID: %d Path: %s Flags: %s", event.ID, event.Path, note)
}

//export stop
func stop(self *C.PyObject) *C.PyObject {
	es.Stop()
	return C.PyArg_BuildNone()
}

//export start
func start(self *C.PyObject) *C.PyObject {
	es.Start()
	ec := es.Events
	go func() {
		for msg := range ec {
			for _, event := range msg {
				callTheCallback(event)
			}
		}
	}()
	return C.PyArg_BuildNone()
}

func main() {}
