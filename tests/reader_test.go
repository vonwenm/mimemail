package mimemail

import (
	"bufio"
	"bytes"
	"code.google.com/p/mahonia"
	"github.com/sunfmin/mimemail"
	"io"
	"io/ioutil"
	"net/mail"
	"net/textproto"
	"os"
	"strings"
	"testing"
)

type utf8reader struct {
}

var charsetaliases = map[string]string{
	"gb2312": "gbk",
}

func (ur *utf8reader) UTF8Reader(charset string, body io.Reader) (r io.Reader, err error) {
	alias := strings.ToLower(charset)
	newname, ok := charsetaliases[alias]
	if ok {
		charset = newname
	}
	r = mahonia.NewDecoder(charset).NewReader(body)
	return
}

var defaultutf8reader = &utf8reader{}

type Case struct {
	Input  string
	Output string
}

var cases = []Case{
	{"=?utf-8?B?SGVsbG8gUERGIOWKoOeCueS4reaWh+WSjOaXpeacrOiqnuOBig==?=\r\n=?utf-8?B?44Gv44KI44GG44GU44GW44GE44G+44GZ?=", `Hello PDF 加点中文和日本語おはようございます`},
	{"=?iso-8859-1?q?J=F6rg_Doe?= <joerg@example.com>", "Jörg Doe <joerg@example.com>"},
	{"=?ISO-8859-1?Q?Andr=E9?= Pirard <PIRARD@vm1.ulg.ac.be>", "André Pirard <PIRARD@vm1.ulg.ac.be>"},
	{"=?ISO-2022-JP?B?RndkOiBFQxskQkwkRn4yWUlKSFYkLDJoTEw+ZTlYRn4yREc9JEskShsoQg==?= =?ISO-2022-JP?B?GyRCJEMkRiQkJGs3bxsoQg==?=", "Fwd: EC未入荷品番が画面上購入可能になっている件"},
}

func TestReader(t *testing.T) {
	for _, c := range cases {
		var err error
		var b []byte
		ir := strings.NewReader(c.Input)

		r := mimemail.NewRFC2047Reader(ir, defaultutf8reader)
		b, err = ioutil.ReadAll(r)
		if err != nil {
			t.Error(err)
		}
		if string(b) != c.Output {
			t.Errorf("expected: %s, but was: %s", c.Output, string(b))
		}
	}
}

func TestISO_8859_1(t *testing.T) {

	var b2 []byte

	f, err := os.Open("iso_8859_1_raw.txt")

	b2, err = ioutil.ReadAll(mimemail.NewISO_8859_1(f))
	if err != nil {
		panic(err)
	}

	expected := []byte("Jörg Doe <joerg@example.com>")

	if bytes.Compare(expected, b2) != 0 {
		t.Errorf("expected: %v, but was: %v", expected, b2)
	}

	// fmt.Println("ISO_8859_1", b)
	// fmt.Println("UTF8      ", []byte("Jörg Doe <joerg@example.com>"))
	// fmt.Println("CONVERTED ", b2)

}

func TestLinelessReader(t *testing.T) {
	r := mimemail.NewLineLessReader(strings.NewReader("aaaa\nbbbbb\nccc\r\nddd"))
	// r := strings.NewReader("aaaabbbbbcccddd")
	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	s := string(b)
	if s != "aaaabbbbbcccddd" {
		t.Error(s)
	}
}

func TestQuotedPrintable(t *testing.T) {
	f, err := os.Open("quoted-printable.txt")
	defer f.Close()
	if err != nil {
		t.Error(err)
	}
	r := mimemail.NewQDecoder(f, false)

	generated := bytes.NewBuffer(nil)
	_, err = io.Copy(generated, r)
	if err != nil {
		t.Error(err)
	}

	var original io.Reader
	original, err = os.Open("original.txt")
	if err != nil {
		t.Error(err)
	}
	originalbuf := bytes.NewBuffer(nil)
	_, err = io.Copy(originalbuf, original)
	if err != nil {
		t.Error(err)
	}
	gen := generated.Bytes()
	org := originalbuf.Bytes()

	for i, _ := range gen {
		if gen[i] != org[i] {
			t.Errorf("\nat %d was: \t\t%s\nexpected:\t\t%s", i, string(gen[i:i+10]), string(org[i:i+10]))
			break
		}
	}
}

type addressCase struct {
	filename          string
	key               string
	expectedAddresses []*mail.Address
}

var addresscases = []addressCase{
	{
		"addresses_japanese.txt",
		"Cc",
		[]*mail.Address{
			{"Anatole Varin", "anatole@theplant.jp"},
			{"Varin Anatole", "a@theplant.jp"},
			{"", "lacoste-dev@theplant.jp"},
			{"畔上淳", "jazegami@fabricant.co.jp"},
			{"Alexandre Miroux", "amiroux@fabricant.co.jp"},
			{"康史 原田", "Y.Harada@trinet-logi.com"},
			{"水本匡俊 水本匡俊", "mmizumoto@fabricant.co.jp"},
			{"松安　賢治／システム開発室　室長", "K.Matsuyasu@trinet-logi.com"},
			{"伊藤　大／IT推進部協力会社", "H.Ito@trinet-logi.com"},
		},
	},
}

func TestAddressList(t *testing.T) {
	for _, c := range addresscases {

		f, err := os.Open(c.filename)
		defer f.Close()
		if err != nil {
			t.Error(err)
		}

		tpr := textproto.NewReader(bufio.NewReader(f))
		var h textproto.MIMEHeader
		h, err = tpr.ReadMIMEHeader()
		if err != nil {
			t.Error(err)
		}
		addresses, err1 := mimemail.AddressList(h, c.key, defaultutf8reader)
		if err1 != nil {
			t.Error(err1)
		}

		if len(addresses) != len(c.expectedAddresses) {
			t.Errorf("wrong length: %d, should be: %d", len(addresses), len(c.expectedAddresses))
			continue
		}

		for i, ad := range c.expectedAddresses {
			if ad.Name != addresses[i].Name {
				t.Errorf("Name wrong at %d, is %s, expected %s", i, addresses[i].Name, ad.Name)
			}

			if ad.Address != addresses[i].Address {
				t.Errorf("Address wrong at %d, is %s, expected %s", i, addresses[i].Address, ad.Address)
			}
		}

	}
}
