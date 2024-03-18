package embed

import (
	"errors"
	"io"
)

type Calculator struct {
	Resolver MathResolver
}

type MathResolver interface {
	Resolve(expression string) (float64, error)
}

func (c Calculator) Process(r io.Reader) (float64, error) {
	expression, err := readOneLine(r)
	if err != nil {
		return 0, err
	}
	if len(expression) == 0 {
		return 0, errors.New("no expression to read")
	}
	answer, err := c.Resolver.Resolve(expression)
	return answer, err
}

func readOneLine(r io.Reader) (string, error) {
	var out []byte
	b := make([]byte, 1)
	for {
		_, err := r.Read(b)
		if err != nil {
			if err == io.EOF {
				return string(out), nil
			}
		}
		if b[0] == '\n' {
			break
		}
		out = append(out, b[0])
	}
	return string(out), nil

	//scanner := bufio.NewScanner(r)
	//for scanner.Scan() {
	//	break
	//}
	//return scanner.Text()
}

//func main() {
//	file, err := os.Open("./interface/embed/expression.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer func(file *os.File) {
//		_ = file.Close()
//	}(file)
//
//	cal := Calculator{
//		Resolver: nil,
//	}
//
//	answer, err := cal.Process(file)
//	if err != nil {
//		log.Print(err)
//	}
//	fmt.Printf("Answer is %f", answer)
//}
