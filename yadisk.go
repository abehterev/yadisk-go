package yadisk

import (
	"encoding/json"
	"errors"
	"sync"
	//"fmt"
	//"github.com/davecgh/go-spew/spew"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	disk *yaDisk
	once sync.Once
)

type yaResList struct {
	P_sort   string        `json:"sort"`
	P_limit  uint64        `json:"limit"`
	P_offset uint64        `json:"offset"`
	P_path   string        `json:"path"`
	P_total  uint64        `json:"total"`
	P_items  []*yaResource `json:"items"`
}

type yaResource struct {
	P_name        string     `json:"name"`
	P_sha256      string     `json:"sha256"`
	P_md5         string     `json:"md5"`
	P_created     string     `json:"created"`
	P_revision    uint64     `json:"revision"`
	P_resource_id string     `json:"resource_id"`
	P_modified    string     `json:"modified"`
	P_media_type  string     `json:"media_type"`
	P_path        string     `json:"path"`
	P_type        string     `json:"type"`
	P_mime_type   string     `json:"mime_type`
	P_size        uint64     `json:"size"`
	P_embed       *yaResList `json:"_embedded"`
}

type yaLink struct {
	P_href      string `json:"href"`
	P_method    string `json:"method"`
	P_templated bool   `json:"templated"`
}

type yaDisk struct {
	oauth   string
	apipath string
	httpCon *http.Client
	mainRes *yaResource
}

func YaDisk(oauth string) *yaDisk {
	if disk == nil {
		once.Do(func() {
			disk = &yaDisk{
				httpCon: &http.Client{},
				oauth:   oauth,
				apipath: "https://cloud-api.yandex.net/v1/disk",
			}
		})
	}
	return disk
}

func (d *yaDisk) ReceiveMainRes() (err error) {
	// Prepare http request
	req, err := http.NewRequest(http.MethodGet, d.apipath+"/resources?path=app:/", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "OAuth "+d.oauth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpCon.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	answer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		//AppResource := &YaResource{}
		//fmt.Printf("answer: (%d)\n%s\n\n", resp.StatusCode, string(answer))
		err := json.Unmarshal(answer, &d.mainRes)
		if err != nil {
			return err
		}
		//spew.Dump(AppResource)
		return nil
		//fmt.Printf("STRUCT:\n%+v\n", AppFolder)
	} else {
		return errors.New("Answer error: " + string(answer))
	}
}

func (d *yaDisk) getUploadPath(name string) (yaresp *yaLink, err error) {
	// Prepare http request
	req, err := http.NewRequest(http.MethodGet, d.apipath+"/resources/upload?path=app:/"+name+"&overwrite=true", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "OAuth "+d.oauth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpCon.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	answer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		yar := &yaLink{}
		err := json.Unmarshal(answer, yar)
		if err != nil {
			return nil, err
		}
		return yar, nil
	} else {
		err = errors.New("GET upload path error: " + string(answer))
		return nil, err
	}
}

func (d *yaDisk) getDownloadPath(name string) (yaresp *yaLink, err error) {
	// Prepare http request
	req, err := http.NewRequest(http.MethodGet, d.apipath+"/resources/download?path=app:/"+name, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "OAuth "+d.oauth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpCon.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	answer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		yar := &yaLink{}
		err := json.Unmarshal(answer, yar)
		if err != nil {
			return nil, err
		}
		return yar, nil
	} else {
		err = errors.New("GET download path error: " + string(answer))
		return nil, err
	}
}

func (d *yaDisk) PutData(name string, reader io.Reader) (err error) {

	upl, err := d.getUploadPath(name)
	if err != nil {
		return err
	}

	//fmt.Printf("upl: %s\n", upl.P_href)

	// Prepare PUT request
	req, err := http.NewRequest(http.MethodPut, upl.P_href, reader)
	if err != nil {
		return err
	}

	resp, err := d.httpCon.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	answer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusCreated {
		return nil
	} else {
		return errors.New("PUT error: (" + resp.Status + ") " + string(answer))
	}
}

func (d *yaDisk) GetCurl(name string) (curlstr string, err error) {
	dwl, err := d.getDownloadPath(name)
	if err != nil {
		return "", err
	}

	str := "curl -L -X " + dwl.P_method + " \"" + dwl.P_href + "\" -o " + name

	return str, nil
}

func (d *yaDisk) GetData(name string, writer io.Writer) (err error) {

	dwl, err := d.getDownloadPath(name)
	if err != nil {
		return err
	}

	//fmt.Printf("upl: %s\n", upl.P_href)

	// Prepare GET request
	req, err := http.NewRequest(http.MethodGet, dwl.P_href, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "OAuth "+d.oauth)

	resp, err := d.httpCon.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		_, err := io.Copy(writer, resp.Body)
		return err
	} else {
		answer, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New("GET error: (" + resp.Status + ") " + string(answer))
	}
}

func (d *yaDisk) DelRes(path string) (err error) {

	// Prepare DELETE request
	req, err := http.NewRequest(http.MethodDelete, d.apipath+"/resources?path=app:/"+path+"&permanently=true", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "OAuth "+d.oauth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpCon.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	answer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		return nil
	} else {
		return errors.New("DELETE error: (" + resp.Status + ") " + string(answer))
	}
}
