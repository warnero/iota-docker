package main

import (
    "fmt"
    "os"
    "errors"
    "bytes"
    "io/ioutil"
    "path/filepath"
    "github.com/asm-products/iota-docker/endpointmgr"
    "github.com/garyburd/redigo/redis"
    "encoding/json"
    "os/exec"
    txttemplate "text/template"
)

const ENDPOINT_ROOT = "user"

type srcTemplateData struct {
    User     string
    Package  string
    SrcType  string
    Src      string
    ErrMsg   string
}

type BuildData struct {
    UserDir      string
    Endpoint     *endpointmgr.Endpoint
    Src          *string
    Endpointmain string
}

func main() {
    epm := endpointmgr.NewEndpointMgr(ENDPOINT_ROOT)
    //INIT OMIT
    c, err := redis.Dial("tcp", "10.20.8.96:6379")
    if err != nil {
        panic(err)
    }
    defer c.Close()

    // retain readability with json
    // serialized, err := json.Marshal(std)

    // if err == nil {
    //     fmt.Println("serialized data: ", string(serialized))
    //     //set
    //     c.Do("SET", "package:iotarocks:function:Hello", serialized)
    // }

    //get
    userfn, err := redis.String(c.Do("GET", "package:iotarocks:function:Hello"))
    if err != nil {
        fmt.Println("key not found")
    }   

    fmt.Println(userfn)
    // Now we need to save the source to a file and try and compile it
    var deserialized srcTemplateData

    err = json.Unmarshal([]byte(userfn), &deserialized)

    if err == nil {
        fmt.Println("deserialized data: ", deserialized.Src)
    }

    userDir := fmt.Sprintf("%s/%s", ENDPOINT_ROOT, deserialized.User)
    userDir, err = filepath.Abs(userDir)
    srcDir := fmt.Sprintf("%s/src/%s/", userDir, deserialized.Package)
    srcFilename := srcDir + "main.go"

    src, err := saveSrc(deserialized.Src, srcFilename, srcDir)
    if err != nil {
        fmt.Sprintf("Error: %s", err)
        return
    }
    msg := fmt.Sprintf("Source file %s saved.\n\n", srcFilename)
    // Build src
    ep, err := doBuild(src, deserialized.Package, userDir, deserialized.User, epm)
    _ = ep
    if err != nil {
        fmt.Sprintf("%sBuild Errors:\n%s\n\nSource:\n%s", msg, err, src)
        return
    }
    fmt.Sprintf("%sBuild Success!\n%s\n", msg, src)
    //ENDINIT OMIT
}

func saveSrc(source string, filename string, pathname string) (src string, err error) {
    fmt.Println("calling saveSrc")
    err = os.MkdirAll(pathname, 0755)
    fmt.Println("created directory", err)
    f, err := os.Create(filename) // FIXME: better validation needs to happen
    fmt.Println("saved file", f, err)
    if err != nil {
        return source, err
    }
    defer f.Close()
    _, err = f.Write([]byte(source))
    if err != nil {
        return source, err
    }
    return source, err
}

func doBuild(src string, packageNameURL string, userDir string, user string, epm *endpointmgr.EndpointMgr) (ep endpointmgr.Endpoint, err error) {
    ep, err = epm.GetEndpointFromSrc(src, user)
    if err != nil {
        return
    }
    bd := &BuildData{
        Endpoint: &ep,
        Src:      &src,
        UserDir:  userDir,
    }
    if ep.Package != packageNameURL {
        msg := fmt.Sprintf("Source package name '%s' does not match URL package's name '%s'", ep.Package, packageNameURL)
        return ep, errors.New(msg)
    }
    fmt.Println("Package:", ep.Package, "Name:", ep.Name)

    err = bd.renderEndpointMain()
    if err != nil {
        return
    }
    err = bd.build()
    return
}

func (bd *BuildData) renderEndpointMain() (err error) {
    tmpl, err := txttemplate.ParseFiles("templates/endpointmain.go")
    if err != nil {
        return
    }
    var b bytes.Buffer
    err = tmpl.Execute(&b, bd.Endpoint)
    if err != nil {
        return
    }
    bd.Endpointmain = b.String()
    return
}

func (bd *BuildData) build() (err error) {
    f, err := ioutil.TempFile("", "endpointmain")
    if err != nil {
        return
    }
    fName := f.Name()
    err = func() (e error) {
        defer f.Close()
        _, err = f.Write([]byte(bd.Endpointmain))
        return
    }()
    if err != nil {
        return
    }
    var buildOut bytes.Buffer
    var buildErr bytes.Buffer
    buildCmd := exec.Command("./buildendpoint.sh", bd.UserDir, bd.Endpoint.Package, fName)
    buildCmd.Stderr = &buildErr
    buildCmd.Stdout = &buildOut
    err = buildCmd.Run()
    fmt.Println("Out:", buildOut.String())
    fmt.Println("Err:", buildErr.String())

    return
}

