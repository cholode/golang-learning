package main

import (
	"fmt"
	"slices"
)

// Index
func Exp1() { //index的使用
	nums := []int{10, 20, 30, 20}

	// 找到：返回第一个匹配元素的下标
	fmt.Println(slices.Index(nums, 20)) // 输出：1

	// 找不到：固定返回 -1
	fmt.Println(slices.Index(nums, 99)) // 输出：-1
}

// Indexfunc
func Exp2() { //indexFunc的使用
	// IndexFunc：按自定义规则找第一个匹配元素的下标，找不到返回 -1
	nums := []int{10, 20, 30}
	fmt.Println(slices.IndexFunc(nums, func(n int) bool { return n > 15 })) // 输出 1
}

// Compare
func Exp3() {
	// Compare：字典序逐位比较两个切片，返回 -1(前者小) / 0(相等) / 1(前者大)
	fmt.Println(slices.Compare([]int{1, 2}, []int{1, 3})) // 输出 -1
}

// Insert
func Exp4() {
	// Insert：在指定索引插入元素，返回新切片（原切片不变）
	s := []int{1, 2, 3}
	fmt.Println(slices.Insert(s, 1, 99, 98)) // 输出 [1 99 98 2 3]
}

// Reverse
func Exp5() {
	// reverse 进行翻转
	s := []int{1, 2, 3}
	slices.Reverse(s[:3])
	fmt.Println(s) // 输出 [3 2 1]
}

// Grow
func Exp6() { // 将剩余容量设置为至少为指定数，可能会更大
	s := []int{1, 2, 3}
	s = slices.Grow(s, 4)
	fmt.Println(cap(s))
	s = slices.Grow(s, 5) // 两个的结果一样
	fmt.Println(cap(s))
}

// s[i:j:k]的引用实验最终的0 <= i <= j <= k <= cap(s)
func Exp7() {
	s := make([]int, 5, 10)
	for i := 0; i < 5; i++ {
		s[i] = i + 1
	}
	s1 := s[2:4:6] // 最后容量是[2:6],6-2=4
	var s2 []int
	s2 = s[2:4]

	s1[1] = 1
	fmt.Println(s)       // [1 2 3 1 5]
	fmt.Println(s1)      // [3 1]
	fmt.Println(s2)      // [3 1]
	fmt.Println(cap(s1)) // 3
	fmt.Println(cap(s2)) // 8
}

// append
func Exp8() {
	s := []int{1, 2, 3, 4, 5, 6}
	s = append(s[:1], 6) // [1 6] append不会把拼接部分开始到后面的部分舍弃，只是缩短了len
	fmt.Println(s)
	fmt.Println(s[:cap(s)]) // [1 6 3 4 5 6] 后面的部分还在
}

// Delete
func Exp9() {
	s := []int{1, 2, 3, 4, 5, 6}
	s = slices.Delete(s, 1, 3) // [1 6] append不会把拼接部分开始到后面的部分舍弃，只是缩短了len
	fmt.Println(s)
}

func main() {
	Exp9()
}
