# Mysql to markdown

This is a simple tool to convert mysql database to markdown.

## Install

```bash
go install github.com/jayecc/mysql2md@latest
```

## Usage

```bash
$ ./mysql2md -h
Usage of mysql2md:
  -ddl                                                                                                                                            
        generate ddl info (default false)                                                                                                         
  -dir string                                                                                                                                     
        directory to save the file (default "./output")                                                                                           
  -dsn string                                                                                                                                     
        database connection string (default "username:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s")
  -whole                                                                                                                                          
        generate whole file (default false)
```

## Example

- Build file contains DDL and define the output directory

```bash
mysql2md -dsn 'root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s' -whole -ddl -dir=.
```

- Build file contains DDL

```bash
mysql2md -dsn 'root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s' -whole -ddl
```

- Build multiple files with DDL

```bash
mysql2md -dsn 'root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s' -ddl
```

- Build multiple files without DDL

```bash
mysql2md -dsn 'root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s'
```

# test tables list

| Name                     | Engine | Create_time               | Collation          | Comment |
|--------------------------|--------|---------------------------|--------------------|---------|
| [member](test.member.md) | InnoDB | 2023-11-13T16:16:52+08:00 | utf8mb4_general_ci | `账户信息`  |

# test.member

> 账户信息

### COLUMNS

| COLUMN_NAME | COLUMN_DEFAULT | IS_NULLABLE | COLLATION_NAME     | COLUMN_TYPE      | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
|-------------|----------------|-------------|--------------------|------------------|------------|----------------|----------------|
| id          |                | NO          |                    | int(10) unsigned | PRI        | auto_increment | ``             |
| nickname    |                | NO          | utf8mb4_general_ci | varchar(30)      | MUL        |                | `昵称`           |

### DDL

```sql
CREATE TABLE `member` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `nickname` varchar(30) NOT NULL COMMENT '昵称',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='账户信息'
```

## License

[MIT](LICENSE)
