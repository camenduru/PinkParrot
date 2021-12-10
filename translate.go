package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"

	"github.com/robertkrimen/otto"
)

var vm = otto.New()

func sM(a otto.Value, TTK ...otto.Value) (otto.Value, error) {
	err := vm.Set("x", a)
	if err != nil {
		return otto.UndefinedValue(), err
	}

	if len(TTK) > 0 {
		_ = vm.Set("internalTTK", TTK[0])
	} else {
		_ = vm.Set("internalTTK", "0")
	}

	result, err := vm.Run(`
		function sM(a) {
			var b;
			if (null !== yr)
				b = yr;
			else {
				b = wr(String.fromCharCode(84));
				var c = wr(String.fromCharCode(75));
				b = [b(), b()];
				b[1] = c();
				b = (yr = window[b.join(c())] || "") || ""
			}
			var d = wr(String.fromCharCode(116))
				, c = wr(String.fromCharCode(107))
				, d = [d(), d()];
			d[1] = c();
			c = "&" + d.join("") + "=";
			d = b.split(".");
			b = Number(d[0]) || 0;
			for (var e = [], f = 0, g = 0; g < a.length; g++) {
				var l = a.charCodeAt(g);
				128 > l ? e[f++] = l : (2048 > l ? e[f++] = l >> 6 | 192 : (55296 == (l & 64512) && g + 1 < a.length && 56320 == (a.charCodeAt(g + 1) & 64512) ? (l = 65536 + ((l & 1023) << 10) + (a.charCodeAt(++g) & 1023),
					e[f++] = l >> 18 | 240,
					e[f++] = l >> 12 & 63 | 128) : e[f++] = l >> 12 | 224,
					e[f++] = l >> 6 & 63 | 128),
					e[f++] = l & 63 | 128)
			}
			a = b;
			for (f = 0; f < e.length; f++)
				a += e[f],
					a = xr(a, "+-a^+6");
			a = xr(a, "+-3^+b+-f");
			a ^= Number(d[1]) || 0;
			0 > a && (a = (a & 2147483647) + 2147483648);
			a %= 1E6;
			return c + (a.toString() + "." + (a ^ b))
		}

		var yr = null;
		var wr = function(a) {
			return function() {
				return a
			}
		}
			, xr = function(a, b) {
			for (var c = 0; c < b.length - 2; c += 3) {
				var d = b.charAt(c + 2)
					, d = "a" <= d ? d.charCodeAt(0) - 87 : Number(d)
					, d = "+" == b.charAt(c + 1) ? a >>> d : a << d;
				a = "+" == b.charAt(c) ? a + d & 4294967295 : a ^ d
			}
			return a
		};
		
		var window = {
			TKK: internalTTK
		};

		sM(x)
	`)
	if err != nil {
		return otto.UndefinedValue(), err
	}

	return result, nil
}

func updateTTK(TTK otto.Value) (otto.Value, error) {
	t := time.Now().UnixNano() / 3600000
	now := math.Floor(float64(t))
	ttk, err := strconv.ParseFloat(TTK.String(), 64)
	if err != nil {
		return otto.UndefinedValue(), err
	}

	if ttk == now {
		return TTK, nil
	}

	resp, err := http.Get(fmt.Sprintf("https://translate.%s", GoogleHost))
	if err != nil {
		return otto.UndefinedValue(), err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return otto.UndefinedValue(), err
	}

	matches := regexp.MustCompile(`tkk:\s?'(.+?)'`).FindStringSubmatch(string(body))
	if len(matches) > 0 {
		v, err := otto.ToValue(matches[0])
		if err != nil {
			return otto.UndefinedValue(), err
		}
		return v, nil
	}

	return TTK, nil
}

func get(text otto.Value, ttk otto.Value) string {
	ttk, err := updateTTK(ttk)
	if err != nil {
		return ""
	}

	tk, err := sM(text, ttk)

	if err != nil {
		return ""
	}
	sTk := strings.Replace(tk.String(), "&tk=", "", -1)
	return sTk

}

var errBadNetwork = errors.New("bad network, please check your internet connection")
var errBadRequest = errors.New("bad request, request on google translate api isn't working")

var GoogleHost = "google.com"

var ttk otto.Value

func init() {
	ttk, _ = otto.ToValue("0")
}

const (
	defaultNumberOfRetries = 2
)

func translate(text, from, to string, withVerification bool, tries int, delay time.Duration) (string, string, error) {
	if tries == 0 {
		tries = defaultNumberOfRetries
	}

	if withVerification {
		if _, err := language.Parse(from); err != nil && from != "auto" {
			log.Println("[WARNING], '" + from + "' is a invalid language, switching to 'auto'")
			from = "auto"
		}
		if _, err := language.Parse(to); err != nil {
			log.Println("[WARNING], '" + to + "' is a invalid language, switching to 'en'")
			to = "en"
		}
	}

	t, _ := otto.ToValue(text)

	urll := fmt.Sprintf("https://translate.%s/translate_a/single", GoogleHost)

	token := get(t, ttk)

	data := map[string]string{
		"client": "gtx",
		"sl":     from,
		"tl":     to,
		"hl":     to,
		// "dt":     []string{"at", "bd", "ex", "ld", "md", "qca", "rw", "rm", "ss", "t"},
		"ie":   "UTF-8",
		"oe":   "UTF-8",
		"otf":  "1",
		"ssel": "0",
		"tsel": "0",
		"kc":   "7",
		"q":    text,
	}

	u, err := url.Parse(urll)
	if err != nil {
		return "", "", nil
	}

	parameters := url.Values{}

	for k, v := range data {
		parameters.Add(k, v)
	}
	for _, v := range []string{"at", "bd", "ex", "ld", "md", "qca", "rw", "rm", "ss", "t"} {
		parameters.Add("dt", v)
	}

	parameters.Add("tk", token)
	u.RawQuery = parameters.Encode()

	var r *http.Response

	for tries > 0 {
		r, err = http.Get(u.String())
		if err != nil {
			if err == http.ErrHandlerTimeout {
				return "", "", errBadNetwork
			}
			return "", "", err
		}

		if r.StatusCode == http.StatusOK {
			break
		}

		if r.StatusCode == http.StatusForbidden {
			tries--
			time.Sleep(delay)
		}
	}

	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", "", err
	}

	var resp []interface{}

	err = json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		return "", "", err
	}

	responseText := ""
	srcText := ""
	for _, obj := range resp[0].([]interface{}) {
		if len(obj.([]interface{})) == 0 {
			break
		}

		t, ok := obj.([]interface{})[0].(string)
		if ok {
			responseText += t
		}
	}

	s, ok := resp[2].(string)
	if ok {
		srcText += s
	}

	return srcText, responseText, nil
}

// TranslationParams is a util struct to pass as parameter to indicate how to translate
type TranslationParams struct {
	From       string
	To         string
	Tries      int
	Delay      time.Duration
	GoogleHost string
}

// Translate translate a text using native tags offer by go language
func Translate(text string, from language.Tag, to language.Tag, googleHost ...string) (string, string, error) {
	if len(googleHost) != 0 && googleHost[0] != "" {
		GoogleHost = googleHost[0]
	}
	src, translated, err := translate(text, from.String(), to.String(), false, 2, 0)
	if err != nil {
		return "", "", err
	}

	return src, translated, nil
}

// TranslateWithParams translate a text with simple params as string
func TranslateWithParams(text string, params TranslationParams) (string, string, error) {
	if params.GoogleHost == "" {
		GoogleHost = "google.com"
	} else {
		GoogleHost = params.GoogleHost
	}
	src, translated, err := translate(text, params.From, params.To, true, params.Tries, params.Delay)
	if err != nil {
		return "", "", err
	}
	return src, translated, nil
}
