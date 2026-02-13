package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func main() {
	templateFile := os.Args[1]  // 模板文件路径
	src := os.Args[2]           // 源文件路径
	output := os.Args[3]        // 输出文件路径

	// 读取模板文件
	b, err := ioutil.ReadFile(templateFile)
	if err != nil {
		panic(err)
	}
	// 读取源文件
	b2, err := ioutil.ReadFile(src)
	if err != nil {
		panic(err)
	}

	// 合并并处理模板
	templated, err := TemplatedString(fmt.Sprintf("%s\n%s", string(b), string(b2)), nil)
	if err != nil {
		panic(err)
	}

	// 写入输出文件
	err = ioutil.WriteFile(output, []byte(templated), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

// TemplatedString 将模板字符串与数据结合生成最终字符串
// t: 模板字符串
// i: 模板数据
func TemplatedString(t string, i interface{}) (string, error) {
	b := bytes.NewBuffer([]byte{})
	tmpl, err := template.New("template").Funcs(sprig.TxtFuncMap()).Parse(t)
	if err != nil {
		return "", err
	}

	err = tmpl.Execute(b, i)

	return b.String(), err
}
