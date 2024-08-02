package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type SearchResults struct {
	ready   bool
	Query   string
	Results []Result
}

type Result struct {
	Name, Description, URL string
}

func WikipediaAPI(request string) (answer []string) {

	//Создаем срез на 3 элемента
	s := make([]string, 3)

	//Отправляем запрос
	if response, err := http.Get(request); err != nil {
		s[0] = "Википедия не отвечает"
	} else {
		defer response.Body.Close()

		//Считываем ответ
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error to read content: %s\n", err)
		}

		//Отправляем данные в структуру
		sr := &SearchResults{}
		if err = json.Unmarshal([]byte(contents), sr); err != nil {
			s[0] = "Что-то пошло не так, попробуйте изменить вопрос."
		}

		//Проверяем не пустая ли наша структура
		if !sr.ready {
			s[0] = "Что-то пошло не так, попробуйте изменить вопрос."
		}

		//Проходим через нашу структуру и отправляем данные в срез с ответом
		for i := range sr.Results {
			s[i] = sr.Results[i].URL
		}
	}

	return s
}

func (sr *SearchResults) UnmarshalJSON(bs []byte) error {
	array := []interface{}{}
	if err := json.Unmarshal(bs, &array); err != nil {
		return err
	}
	sr.Query = array[0].(string)
	for i := range array[1].([]interface{}) {
		sr.Results = append(sr.Results, Result{
			array[1].([]interface{})[i].(string),
			array[2].([]interface{})[i].(string),
			array[3].([]interface{})[i].(string),
		})
	}
	return nil
}
