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
	// 1. IndexFunc：按自定义规则找第一个匹配元素的下标，找不到返回 -1
	nums := []int{10, 20, 30}
	fmt.Println(slices.IndexFunc(nums, func(n int) bool { return n > 15 })) // 输出 1
}

// Compare
func Exp3() {
	// 2. Compare：字典序逐位比较两个切片，返回 -1(前者小) / 0(相等) / 1(前者大)
	fmt.Println(slices.Compare([]int{1, 2}, []int{1, 3})) // 输出 -1
}

// Insert
func Exp4() {
	// 3. Insert：在指定索引插入元素，返回新切片（原切片不变）
	s := []int{1, 2, 3}
	fmt.Println(slices.Insert(s, 1, 99, 98)) // 输出 [1 99 98 2 3]
}

func main() {

}
