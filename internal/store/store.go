package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Animal struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Species string `json:"species"`
	Breed string `json:"breed"`
	Age int `json:"age"`
	Weight int `json:"weight"`
	Status string `json:"status"`
	Location string `json:"location"`
	Notes string `json:"notes"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"paddock.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS animals(id TEXT PRIMARY KEY,name TEXT NOT NULL,species TEXT DEFAULT '',breed TEXT DEFAULT '',age INTEGER DEFAULT 0,weight INTEGER DEFAULT 0,status TEXT DEFAULT 'active',location TEXT DEFAULT '',notes TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Animal)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO animals(id,name,species,breed,age,weight,status,location,notes,created_at)VALUES(?,?,?,?,?,?,?,?,?,?)`,e.ID,e.Name,e.Species,e.Breed,e.Age,e.Weight,e.Status,e.Location,e.Notes,e.CreatedAt);return err}
func(d *DB)Get(id string)*Animal{var e Animal;if d.db.QueryRow(`SELECT id,name,species,breed,age,weight,status,location,notes,created_at FROM animals WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Species,&e.Breed,&e.Age,&e.Weight,&e.Status,&e.Location,&e.Notes,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Animal{rows,_:=d.db.Query(`SELECT id,name,species,breed,age,weight,status,location,notes,created_at FROM animals ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Animal;for rows.Next(){var e Animal;rows.Scan(&e.ID,&e.Name,&e.Species,&e.Breed,&e.Age,&e.Weight,&e.Status,&e.Location,&e.Notes,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *Animal)error{_,err:=d.db.Exec(`UPDATE animals SET name=?,species=?,breed=?,age=?,weight=?,status=?,location=?,notes=? WHERE id=?`,e.Name,e.Species,e.Breed,e.Age,e.Weight,e.Status,e.Location,e.Notes,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM animals WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM animals`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]Animal{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (name LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    if v,ok:=filters["status"];ok&&v!=""{where+=" AND status=?";args=append(args,v)}
    rows,_:=d.db.Query(`SELECT id,name,species,breed,age,weight,status,location,notes,created_at FROM animals WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []Animal;for rows.Next(){var e Animal;rows.Scan(&e.ID,&e.Name,&e.Species,&e.Breed,&e.Age,&e.Weight,&e.Status,&e.Location,&e.Notes,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    rows,_:=d.db.Query(`SELECT status,COUNT(*) FROM animals GROUP BY status`)
    if rows!=nil{defer rows.Close();by:=map[string]int{};for rows.Next(){var s string;var c int;rows.Scan(&s,&c);by[s]=c};m["by_status"]=by}
    return m
}
