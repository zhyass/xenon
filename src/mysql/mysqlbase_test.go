/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package mysql

import (
	"fmt"
	"model"
	"testing"

	"config"
	"xbase/xlog"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var mysqlbase = new(MysqlBase)

func TestMysqlBasePing(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	query := "SHOW SLAVE STATUS"
	columns := []string{"Master_Log_File",
		"Read_Master_Log_Pos",
		"Relay_Master_Log_File",
	}
	mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
		"147",
		"mysql-bin.000001",
	)

	mock.ExpectQuery(query).WillReturnRows(mockRows)
	pe, err := mysqlbase.Ping(db)
	assert.Nil(t, err)

	want := "mysql-bin.000001"
	got := pe.Relay_Master_Log_File
	assert.Equal(t, want, got)
}

func TestMysqlBaseGetSlaveGTIDGotZeroRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	query := "SHOW SLAVE STATUS"
	columns := []string{"Master_Log_File",
		"Read_Master_Log_Pos",
		"Retrieved_Gtid_Set",
		"Executed_Gtid_Set",
		"Slave_IO_Running",
		"Slave_SQL_Running",
		"Seconds_Behind_Master",
		"Last_Error",
		"Slave_SQL_Running_State",
	}

	want := model.GTID{}
	mockRows := sqlmock.NewRows(columns).AddRow("", "", "", "", "", "", "", "", "")
	mock.ExpectQuery(query).WillReturnRows(mockRows)
	got, err := mysqlbase.GetSlaveGTID(db)
	assert.Nil(t, err)
	assert.Equal(t, want, *got)
}

func TestMysqlBaseGetSlaveGTID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	query := "SHOW SLAVE STATUS"
	columns := []string{"Master_Log_File",
		"Read_Master_Log_Pos",
		"Retrieved_Gtid_Set",
		"Executed_Gtid_Set",
		"Slave_IO_Running",
		"Slave_SQL_Running",
		"Seconds_Behind_Master",
		"Last_Error",
		"Slave_SQL_Running_State",
	}

	want := model.GTID{Master_Log_File: "mysql-bin.000001",
		Read_Master_Log_Pos:     147,
		Retrieved_GTID_Set:      "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		Executed_GTID_Set:       "84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
		Slave_IO_Running:        true,
		Slave_IO_Running_Str:    "Yes",
		Slave_SQL_Running:       true,
		Slave_SQL_Running_Str:   "Yes",
		Seconds_Behind_Master:   "11",
		Last_Error:              "",
		Slave_SQL_Running_State: "Slave has read all relay log; waiting for the slave I/O thread to update it",
	}

	mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
		"147",
		"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
		"Yes",
		"Yes",
		"11",
		"",
		"Slave has read all relay log; waiting for the slave I/O thread to update it",
	)

	mock.ExpectQuery(query).WillReturnRows(mockRows)
	got, err := mysqlbase.GetSlaveGTID(db)
	assert.Nil(t, err)
	assert.Equal(t, want, *got)
}

func TestMysqlBaseGetMasterGTID(t *testing.T) {
	// 1. mysql up
	{
		db, mock, err := sqlmock.New()
		assert.Nil(t, err)
		defer db.Close()

		query := "SHOW MASTER STATUS"
		columns := []string{"File",
			"Position",
			"Binlog_Do_DB",
			"Binlog_Ignore_DB",
			"Executed_Gtid_Set",
		}

		want := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos:     147,
			Retrieved_GTID_Set:      "",
			Executed_GTID_Set:       "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Slave_IO_Running:        true,
			Slave_SQL_Running:       true,
			Seconds_Behind_Master:   "0",
			Last_Error:              "",
			Slave_SQL_Running_State: "",
		}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"",
			"",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		)

		mock.ExpectQuery(query).WillReturnRows(mockRows)
		got, err := mysqlbase.GetMasterGTID(db)
		assert.Nil(t, err)
		assert.Equal(t, want, *got)
	}
}

func TestGetUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	query := "SELECT @@SERVER_UUID"
	columns := []string{"@@SERVER_UUID"}
	mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	want := "84030605-66aa-11e6-9465-52540e7fd51c"

	got, _ := mysqlbase.GetUUID(db)
	assert.Equal(t, want, got)
}

func TestMysqlBaseChangeMasterToCommand(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	want := []string{
		`CHANGE MASTER TO
  MASTER_HOST = 'localhost',
  MASTER_PORT = 123,
  MASTER_USER = 'username',
  MASTER_PASSWORD = 'password',
  MASTER_AUTO_POSITION = 1`}

	master := model.Repl{Master_Host: "localhost",
		Master_Port:   123,
		Repl_User:     "username",
		Repl_Password: "password"}

	got := mysqlbase.changeMasterToCommands(&master)
	assert.Equal(t, want, got)
}

func TestMysqlBaseChangeMasterTo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{"STOP SLAVE",
		`CHANGE MASTER TO MASTER_HOST = 'localhost', MASTER_PORT = 123, MASTER_USER = 'username', MASTER_PASSWORD = 'password', MASTER_AUTO_POSITION = 1`,
		"START SLAVE",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))

	master := model.Repl{Master_Host: "localhost",
		Master_Port:   123,
		Repl_User:     "username",
		Repl_Password: "password"}
	err = mysqlbase.ChangeMasterTo(db, &master)
	assert.Nil(t, err)
}

func TestMysqlBaseChangeToMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{"STOP SLAVE",
		"RESET SLAVE ALL",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.ChangeToMaster(db)
	assert.Nil(t, err)
}

func TestMysqlBaseSlaveIOThread(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"START SLAVE IO_THREAD",
		"STOP SLAVE IO_THREAD",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.StartSlaveIOThread(db)
	assert.Nil(t, err)

	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.StopSlaveIOThread(db)
	assert.Nil(t, err)
}

func TestMysqlBaseReadOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"SET GLOBAL read_only = 1",
		"SET GLOBAL super_read_only = 1",
		"SET GLOBAL read_only = 0",
		"SET GLOBAL super_read_only = 0",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.SetReadOnly(db, true)
	assert.Nil(t, err)

	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[3]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.SetReadOnly(db, false)
	assert.Nil(t, err)
}

func TestMysqlBaseSetGlobalVar(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"SET GLOBAL read_only = 1",
		"SET GLOBAL gtid_mode = 'ON'",
		"XET GLOBAL gtid_mode = 'ON'",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.SetGlobalSysVar(db, queryList[0])
	assert.Nil(t, err)

	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.SetGlobalSysVar(db, queryList[1])
	assert.Nil(t, err)

	err = mysqlbase.SetGlobalSysVar(db, queryList[2])
	want := "[XET GLOBAL gtid_mode = 'ON'].must.be.startwith:SET GLOBAL"
	got := err.Error()
	assert.Equal(t, want, got)
}

func TestMysqlBaseResetMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"RESET MASTER",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.ResetMaster(db)
	assert.Nil(t, err)
}

func TestMysqlBasePurgeBinlogsTo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"PURGE BINARY LOGS TO 'mysql-bin.000032'",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.PurgeBinlogsTo(db, "mysql-bin.000032")
	assert.Nil(t, err)
}

func TestMysqlBaseSemiMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"SET GLOBAL rpl_semi_sync_master_enabled=ON",
		"SET GLOBAL rpl_semi_sync_master_enabled=OFF",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.EnableSemiSyncMaster(db)
	assert.Nil(t, err)

	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.DisableSemiSyncMaster(db)
	assert.Nil(t, err)
}

func TestMysqlBaseSemiMasterDefault(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	queryList := []string{
		"SET GLOBAL rpl_semi_sync_master_timeout=default",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysqlbase.SetSemiSyncMasterDefault(db)
	assert.Nil(t, err)
}

func TestCheckUserExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "SELECT User FROM mysql.user WHERE User = 'xx'"
	columns := []string{"User"}
	want := true
	mockRows := sqlmock.NewRows(columns).AddRow(want)
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	exists, err := mysql.CheckUserExists("xx")
	assert.Nil(t, err)

	got := exists
	assert.Equal(t, want, got)
}

func TestCreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "CREATE USER `xx` IDENTIFIED BY 'xxx'"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.CreateUser("xx", "xxx")
	assert.Nil(t, err)
}

func TestCreateUserWithPrivileges(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "GRANT ALTER , ALTER ROUTINE ON test.* TO `xx`@'127.0.0.1' IDENTIFIED BY 'pwd'"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.CreateUserWithPrivileges("xx", "pwd", "test", "*", "127.0.0.1", "ALTER , ALTER ROUTINE", "NO")
	assert.Nil(t, err)

	query = "GRANT ALTER , ALTER ROUTINE ON test.* TO `xx`@'%' IDENTIFIED BY 'pwd' REQUIRE X509"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.CreateUserWithPrivileges("xx", "pwd", "test", "*", "%", "ALTER , ALTER ROUTINE", "YES")
	assert.Nil(t, err)
}

func TestGetUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "SELECT User, Host, Super_priv FROM mysql.user"
	columns := []string{"User", "Host", "Super_priv"}
	want := []model.MysqlUser{
		{User: "user1",
			Host:      "localhost",
			SuperPriv: "N"},
		{User: "root",
			Host:      "localhost",
			SuperPriv: "Y"},
	}

	mockRows := sqlmock.NewRows(columns).AddRow("user1", "localhost", "N").AddRow("root", "localhost", "Y")
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	got, err := mysql.GetUser()
	assert.Nil(t, err)

	assert.Equal(t, want, got)
}

func TestDropUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "DROP USER `xx`@'127.0.0.1'"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.DropUser("xx", "127.0.0.1")
	assert.Nil(t, err)
}

func TestCreateReplUserWithoutBinlog(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	queryList := []string{
		"SET sql_log_bin=0",
		"CREATE USER `repl` IDENTIFIED BY 'replpwd'",
		"GRANT REPLICATION SLAVE,REPLICATION CLIENT ON *.* TO `repl`",
		"SET sql_log_bin=1",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[3]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.CreateReplUserWithoutBinlog("repl", "replpwd")
	assert.Nil(t, err)
}

func TestCreateReplUserWithoutBinlogErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	queryList := []string{
		"SET sql_log_bin=0",
		"CREATE USER `repl` IDENTIFIED BY 'replpwd'",
		"GRANT REPLICATION SLAVE,REPLICATION CLIENT ON *.* TO `repl`",
		"SET sql_log_bin=1",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[2]).WillReturnError(fmt.Errorf("ERROR 1045 (28000): Access denied for user 'repl'@'%%'"))
	mock.ExpectExec(queryList[3]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.CreateReplUserWithoutBinlog("repl", "replpwd")
	want := "ERROR 1045 (28000): Access denied for user 'repl'@'%'"
	got := err.Error()
	assert.Equal(t, want, got)
}

func TestChangeUserPasswd(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "ALTER USER `xx`@'localhost' IDENTIFIED BY 'xxx'"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.ChangeUserPasswd("xx", "localhost", "xxx")
	assert.Nil(t, err)
}

func TestGrantAllPrivileges(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	var query string
	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query = "GRANT ALL ON *.* TO `xx` IDENTIFIED BY 'xx' WITH GRANT OPTION"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.GrantAllPrivileges("xx", "xx", "NO")
	assert.Nil(t, err)

	query = "GRANT ALL ON *.* TO `xx` IDENTIFIED BY 'xx' REQUIRE X509 WITH GRANT OPTION"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.GrantAllPrivileges("xx", "xx", "YES")
	assert.Nil(t, err)
}

func TestGrantNormalPrivileges(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "GRANT ALTER,ALTER ROUTINE,CREATE,CREATE ROUTINE,CREATE TEMPORARY TABLES,CREATE VIEW,DELETE,DROP,EXECUTE,EVENT,INDEX,INSERT,LOCK TABLES,PROCESS,RELOAD,SELECT,SHOW DATABASES,SHOW VIEW,UPDATE,TRIGGER,REFERENCES,REPLICATION SLAVE,REPLICATION CLIENT ON *.* TO `xx`"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.GrantNormalPrivileges("xx")
	assert.Nil(t, err)
}

func TestGrantReplicationPrivileges(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	query := "GRANT REPLICATION SLAVE,REPLICATION CLIENT ON *.* TO `xx`"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.GrantReplicationPrivileges("xx")
	assert.Nil(t, err)
}

func TestGrantUserPrivileges(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, log)
	mysql.db = db

	// ok
	{
		user := "xx"
		passwd := "pwd"
		database := "db"
		table := "table1"
		host := "192.168.0.1"
		privs := "SELECT"
		ssl := "NO"
		query := "GRANT SELECT ON db.table1 TO `xx`@'192.168.0.1' IDENTIFIED BY 'pwd'"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
		err = mysql.CreateUserWithPrivileges(user, passwd, database, table, host, privs, ssl)
		assert.Nil(t, err)
	}

	// ok
	{
		user := "xx"
		passwd := "pwd"
		database := "db"
		table := "table1"
		host := "192.168.0.1"
		privs := "SELECT,"
		ssl := "YES"
		query := "GRANT SELECT ON db.table1 TO `xx`@'192.168.0.1' IDENTIFIED BY 'pwd' REQUIRE X509"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
		err = mysql.CreateUserWithPrivileges(user, passwd, database, table, host, privs, ssl)
		assert.Nil(t, err)
	}

	// error
	{
		user := "xx"
		passwd := "pwd"
		database := "db"
		table := "table1"
		host := "192.168.0.1"
		privs := "SELECT,GRANT OPTION"
		ssl := "X509"
		query := "GRANT SELECT ON db.table1 TO `xx`@'192.168.0.1' IDENTIFIED BY 'pwd' REQUIRE X509"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
		err = mysql.CreateUserWithPrivileges(user, passwd, database, table, host, privs, ssl)
		want := "can't create user[xx] with privileges[GRANT OPTION]"
		got := err.Error()
		assert.Equal(t, want, got)
	}
}
