package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})
	for _, job := range jobs {
		wg.Add(1)
		out := make(chan interface{}, 1)
		go jobWorker(job, in, out, wg)
		in = out
	}

	wg.Wait()
}

func jobWorker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)

	job(in, out)
}

/*
SingleHash -
*/
func SingleHash(in, out chan interface{}) {
	mtx := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for i := range in {
		wg.Add(1)
		data := strconv.Itoa(i.(int))

		go (func() {
			defer wg.Done()

			mtx.Lock()
			dataMd5 := DataSignerMd5(data)
			mtx.Unlock()

			crc32Ch := make(chan string)
			go (func(out chan string) {
				out <- DataSignerCrc32(data)
			})(crc32Ch)
			crc32dataMd5 := DataSignerCrc32(dataMd5)
			crc32data := <-crc32Ch

			fmt.Printf("%s SingleHash data %s\n", data, data)
			fmt.Printf("%s SingleHash md5(data) %s\n", data, dataMd5)
			fmt.Printf("%s SingleHash crc32(md5(data)) %s\n", data, crc32dataMd5)
			fmt.Printf("%s SingleHash crc32(data) %s\n", data, crc32data)
			fmt.Printf("%s SingleHash result %s\n", data, crc32data+"~"+crc32dataMd5)

			out <- crc32data + "~" + crc32dataMd5
		})()
	}

	wg.Wait()
}

/*
MultiHash -
*/
func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for i := range in {
		wg.Add(1)

		data := i.(string)

		go (func() {
			defer wg.Done()

			mx := sync.Mutex{}
			crc32Wg := sync.WaitGroup{}
			parts := make([]string, 6)

			for i := 0; i <= 5; i++ {
				crc32Wg.Add(1)

				go (func(index int) {
					defer crc32Wg.Done()
					part := DataSignerCrc32(strconv.Itoa(index) + data)

					mx.Lock()
					parts[index] = part
					fmt.Printf("%s MultiHash: crc32(th+step1)) %d %s\n", data, index, part)
					mx.Unlock()
				})(i)
			}

			crc32Wg.Wait()

			result := strings.Join(parts, "")

			fmt.Printf("%s MultiHash result: %s\n", data, result)

			out <- result

		})()
	}

	wg.Wait()
}

/*
CombineResults -
*/
func CombineResults(in, out chan interface{}) {
	var results []string

	for i := range in {
		results = append(results, i.(string))
	}

	sort.Strings(results)
	result := strings.Join(results, "_")

	fmt.Printf("CombineResults \n%s\n", result)

	out <- result
}
