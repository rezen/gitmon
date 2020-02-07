package gitmon

import (
	"github.com/jinzhu/gorm"
	// "github.com/asaskevich/EventBus"

)

type State struct {
	DB *gorm.DB
	Emitter *BetterBus
}

func (s State) Has(name string) bool {
	count := 0
	s.DB.Table("state").Where("name = ?", name).Count(&count)
	return count > 0
}

// @todo make json?
func (s State) Get(name string) string {
	row := s.DB.Table("state").Where("name = ?", name).Select("value").Row()
	var value string
	row.Scan(&value)
	return value
}

func (s State) GetPrefix(name string) map[string]string {
	result := map[string]string{}
	rows, err := s.DB.Table("state").Order("name", true).Where("name LIKE ?", name + "%").Select("name, value").Rows()
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var name string
		var value string
		rows.Scan(&name, &value)
		result[name] = value
	}

	return result
}

func (s State) GetSuffix(name string) map[string]string {
	result := map[string]string{}
	rows, err := s.DB.Table("state").Order("name", true).Where("name LIKE ?", "%" + name).Select("name, value").Rows()
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var name string
		var value string
		rows.Scan(&name, &value)
		result[name] = value
	}

	return result
}

func (s State) Put(name string, value string) {
	if s.Has(name) {
		s.DB.Exec("UPDATE state SET value = ? WHERE name = ?", value, name)
	} else {
		s.DB.Exec("INSERT INTO state (name, value) VALUES (?, ?)", name, value)
	} 
}

func (s State) Delete(name string) error {
	s.DB.Exec("DELETE FROM state where name = ?", name)
	return nil
}

func (s State) DeleteSuffix(name string) error {
	s.DB.Exec("DELETE FROM state where name LIKE ?", "%" + name )
	return nil
}

func (s State) DeletePrefix(name string) error {
	s.DB.Exec("DELETE FROM state where name LIKE ?", name + "%" )
	return nil
}