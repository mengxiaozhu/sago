# SAGO (SQL Assistant for **GO**)
## HOW TO USE
1. Define DAO struct
    ```go
        type UserDao struct{
            DB *sql.DB
            FindByName(name string)(*User,error)
        }
    ```

2. Write SQL in XML
    ```xml
        <sago>
            <table>user</table>
            <type>UserDao</type>
            <select name="FindByName" args="name">
                select {{.fields}} from {{.table}} where `name` = {{arg .name}}
            </select>
        </sago>
    ```

3. Create sago manager,Scan XMLs and Mapped to DAO object
    ```go
        dao:=&UserDao{
            DB:db,
        }
        s:=sago.New()
        s.ScanDir("./xmlsDirPath")
        s.Map(dao)
        dao.FindByName("foo")
    ```
