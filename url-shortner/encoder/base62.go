package encoder
const base62Chars= "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
func Encode(num uint64)string{
	if num == 0{
		return string(base62Chars[0])
	}

	res:=""
	for num>0{
		remainder:=num%62
		res=string(base62Chars[remainder])+res
		num/=62
	}
	return res
}