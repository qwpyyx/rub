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

## 阿里云短信配置说明

1. 登录阿里云控制台，开通短信服务并完成实名认证、签名和模板审核。模板变量需要包含 `name`（用户姓名）、`date`（预约日期）、`time`（预约时间），与代码中发送的 `TemplateParam` 字段一致。
2. 在「访问控制」中新建或使用已有的 AccessKey，记录 `AccessKeyId` 和 `AccessKeySecret`。
3. 进入短信服务控制台查看或创建短信签名（SignName）和短信模板（TemplateCode）。
4. 在运行程序前设置以下环境变量：
   ```bash
   export ALIYUN_SMS_ACCESS_KEY_ID="你的AccessKeyId"
   export ALIYUN_SMS_ACCESS_KEY_SECRET="你的AccessKeySecret"
   export ALIYUN_SMS_SIGN_NAME="短信签名"
   export ALIYUN_SMS_TEMPLATE_CODE="模板CODE"
   # 可选，默认 cn-hangzhou
   export ALIYUN_SMS_REGION_ID="cn-hangzhou"
   ```
5. 确保用户信息中填有手机号（在 Web 表单或配置文件中添加），短信才会发送成功。