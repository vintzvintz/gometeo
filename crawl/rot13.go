package crawl

import (
	"io"
	"strings"
	//"bufio"
	"unicode/utf8"
	"fmt"
)

type rot13Reader struct {
	r *strings.Reader
	curRune []byte
}
/*
func Teste() {
	s := strings.NewReader("Lbh àÏ penpxrq pbqr!")
	r := rot13Reader{s}
	io.Copy(os.Stdout, &r)
    fmt.Println("----")
}
*/
func rot13(r rune) rune {
	ret := r
	switch {
	case r >= 'A' && r <= 'Z':
		ret = 'A' + (r-'A'+13)%26
	case r >= 'a' && r <= 'z':
		ret = 'a' + (r-'a'+13)%26
	}
	return ret
}

func writeCurRune(buf, curRune []byte ) ( remaining []byte ) {
	for i := len(buf); (i<cap(buf) && len(curRune)>0); i++ {
		buf[i] = curRune[0]
		curRune = curRune[1:]
	}
	return buf
}


func (r13 rot13Reader) Read(b []byte) (int, error) {

	fmt.Printf("Read() len(b)=%d cap(b)=%d ", len(b), cap(b) )
	//rr := bufio.NewReader( r13.r )
	//var bytes_read int = 0

	// b est un slice rempli de 0 et de longueur inconnue mais égale à la capacité
	// reset b pour utiliser AppendRune
	var buf []byte = b[:0]

	// commence par vider les bytes restants du Read() précédent
	buf = writeCurRune( buf, r13.curRune )
	r13.curRune = make( []byte,4 )

	// traite les runes suivantes
	for {
		ch, _, err := r13.r.ReadRune()
		if err == nil {
			ch13 := rot13(ch)
			nb := utf8.EncodeRune(r13.curRune, ch13)
			buf = writeCurRune( buf, r13.curRune[0:nb] )
		}
		if (len(buf) == cap(buf)) || (err != nil)  {
			fmt.Printf("read=%d err=%q\n", len(buf), err )
    		return len(buf), err
		}
    }
}


func Rot13( in string) string {
	fmt.Printf("Conversion ROT13 de %s\n", in)
	sr := strings.NewReader( in )
	r := rot13Reader{
		r:sr,
		curRune: make([]byte, 0, 4),
	}
	if b, err := io.ReadAll(r); err == nil {
		return string(b)
	}
	fmt.Printf("Erreur ROT13 sur %s", in)
	return in
}