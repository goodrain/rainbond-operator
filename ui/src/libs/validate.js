/* 合法uri */
export function validateURL (textval) {
  const urlregex = /^(https?|ftp):\/\/([a-zA-Z0-9.-]+(:[a-zA-Z0-9.&%$-]+)*@)*((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])){3}|([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.(com|edu|gov|int|mil|net|org|biz|arpa|info|name|pro|aero|coop|museum|[a-zA-Z]{2}))(:[0-9]+)*(\/($|[a-zA-Z0-9.,?'\\+&%$#=~_-]+))*$/
  return urlregex.test(textval)
}

/* 小写字母 */
export function validateLowerCase (str) {
  const reg = /^[a-z]+$/
  return reg.test(str)
}

/* 大写字母 */
export function validateUpperCase (str) {
  const reg = /^[A-Z]+$/
  return reg.test(str)
}

/* 大小写字母 */
export function validateAlphabets (str) {
  const reg = /^[A-Za-z]+$/
  return reg.test(str)
}

/* 手机号 */
export function validatPhone (str) {
  const reg = /^1[3456789]\d{9}$/
  return reg.test(str)
}

/* 账号 */
export function validateAccount (str) {
  const reg = /^[a-zA-Z\d]\w{3,19}[a-zA-Z\d]$/
  return reg.test(str)
}

/* Email地址 */
export function validateEmail (str) {
  const reg = /^[\w-]+(\.[\w-]+)*@([\w-]+\.)+[a-zA-Z]+$/
  return reg.test(str)
}

/* 域名加端口校验 */
export function validateDomainPort (str) {
  const reg = /(\w+)+(\:([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5]))$/
  return reg.test(str)
}
/* 域名校验 */
export function validateDomain (str) {
  const reg = /^(?=.*\w).{3,255}$/
  return reg.test(str)
}

/* ip加端口 */
export function validateIpPort (str) {
  const reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\:([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5])$/
  return reg.test(str)
}

/* ip */
export function validateIp (str) {
  const reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/
  return reg.test(str)
}

/* 企业名称 */
export function validateEnterpriseName (str) {
  const reg = /^(?!_|\-)(?!.*?_$)(?!.*?-$)[a-zA-Z0-9_\-\u4e00-\u9fa5]+$/
  return reg.test(str)
}
