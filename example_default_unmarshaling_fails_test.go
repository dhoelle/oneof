package oneof_test

// Note: This example is commented out because, as of
// https://github.com/go-json-experiment/json/commit/2e55bd4e08b08427ba10066e9617338e1f113c53,
// json.Unmarshal sometimes returns different errors from the same inputs:
//
//   json: cannot unmarshal JSON number into Go value of type fmt.Stringer: cannot derive concrete type for non-empty interface
//
// vs
//
//   unable to unmarshal JSON number into Go value of type fmt.Stringer: cannot derive concrete type for non-empty interface

// func Example_interfaceFails() {
// 	var s1 fmt.Stringer = crypto.SHA256
// 	b, _ := json.Marshal(s1) // b == []byte("5")

// 	var s2 fmt.Stringer
// 	err := json.Unmarshal(b, &s2)
// 	fmt.Println(err.Error())
// 	// Output:
// 	// json: cannot unmarshal JSON number into Go value of type fmt.Stringer: cannot derive concrete type for non-empty interface
// }
