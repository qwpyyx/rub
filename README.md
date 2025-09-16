# 开始运行前修改
1. _WEU, MOD_AUTH_CAS, StudentId, StudentName
2. execRub()里面的httpRequestDHID()函数日期相关参数

## 定时运行
go run main.go

## 直接运行
go run main.go -d

## 只抢第一个
go run main.go -f

## 只抢第二个
go run main.go -s

> 以上运行参数也可以组合使用，如直接运行抢第二个：go run main.go -d -s