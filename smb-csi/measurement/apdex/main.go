package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	v2 "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
import "k8s.io/client-go/tools/clientcmd"

var timesPVC = make([]time.Duration, 0)
var timesPod = make([]time.Duration, 0)
var totalPVCTime time.Duration
var totalPodTime time.Duration
var totalTime time.Duration

var (
	driver = 		 flag.String("driver","smb-simple","Name of the used driver")
	master = 		 flag.String("master","https://192.168.49.2:8443","URL of the Kubernetes Master")
	config = 		 flag.String("config","/home/stefan/.kube/config","Path to Kubernetes Config (usually $HOME/.kube/config)")
	namespace = 	 flag.String("namespace","default","Namespace for creating PVCs and Pods")
	volumesCount = 	 flag.String("volumesCount", "50", "Number of Volumes to be generated")
	msToleratedPVC = flag.String("msToleratedPVC", "0", "Number of millis to which the duration of creating PVCs will be tolerated")
	msToleratedPod = flag.String("msToleratedPod", "0", "Number of millis to which the duration of creating Pods will be tolerated")
	msFailurePVC = 	 flag.String("msFailurePVC", "0", "Number of millis to which the duration of creating PVCs will be counted as failure")
	msFailurePod = 	 flag.String("msFailurePod", "0", "Number of millis to which the duration of creating Pods will be counted as failure")
)

var podNamePattern string
var pvcNamePattern string

func main() {

	flag.Parse()
	if *driver == "" {
		println("No valid driverName specified")
		os.Exit(1)
	}
	if *config == "" {
		println("Path to kube config must be specified")
		os.Exit(1)
	}
	volumesCountParsed, err := strconv.Atoi(*volumesCount)
	if err != nil {
		println("volumesCount must be a number")
		os.Exit(1)
	}
	msToleratedPVCParsed, err := strconv.ParseInt(*msToleratedPVC, 10, 64)
	if err != nil {
		println("msToleratedPVC must be a number")
		os.Exit(1)
	}
	msFailurePVCParsed, err := strconv.ParseInt(*msFailurePVC, 10, 64)
	if err != nil {
		println("msFailurePVC must be a number")
		os.Exit(1)
	}

	msToleratedPodParsed, err := strconv.ParseInt(*msToleratedPod, 10, 64)
	if err != nil {
		println("msToleratedPod must be a number")
		os.Exit(1)
	}
	msFailurePodParsed, err := strconv.ParseInt(*msFailurePod, 10, 64)
	if err != nil {
		println("msFailurePod must be a number")
		os.Exit(1)
	}

	podNamePattern = *driver + "-pod-"
	pvcNamePattern = *driver + "-pvc-"

	conf, err := clientcmd.BuildConfigFromFlags(*master, *config)
	if err != nil {
		println(err)
	}

	client, err := kubernetes.NewForConfig(conf)
	pvcClient := client.CoreV1().PersistentVolumeClaims(*namespace)
	podClient := client.CoreV1().Pods(*namespace)
	totalStartTime := time.Now()

	MeasurePVCTimesThreaded(pvcClient, volumesCountParsed)
	MeasurePodTimesThreaded(podClient, volumesCountParsed)

	totalEndTime := time.Now()
	totalTime = totalEndTime.Sub(totalStartTime)

	fmt.Printf("TimesPVC: %+v\n", timesPVC)
	fmt.Printf("TimesPod: %+v\n", timesPod)
	fmt.Printf("Total Time PVC: %+v\n", totalPVCTime)
	fmt.Printf("Total Time Pod: %+v\n", totalPodTime)
	fmt.Printf("Total Time: %+v\n", totalTime)

	generateChart(timesPVC, "Time for provisioning and binding a volume (" + *driver + ")", "barPVC.html", msToleratedPVCParsed, msFailurePVCParsed)
	generateChart(timesPod, "Time for binding volumes to a client (" + *driver + ")","barPod.html", msToleratedPodParsed, msFailurePodParsed)
}

func MeasurePVCTimesThreaded(pvcClient v2.PersistentVolumeClaimInterface, count int) {
	timesPVC = make([]time.Duration, count)
	totalStartTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go PVCworker(&wg, pvcClient, i)
	}

	wg.Wait()
	totalEndTime := time.Now()
	totalPVCTime = totalEndTime.Sub(totalStartTime)
}

func PVCworker(wg *sync.WaitGroup, pvcClient v2.PersistentVolumeClaimInterface, i int) {
	defer wg.Done()
	var pvcName = pvcNamePattern + strconv.Itoa(i)
	start := time.Now()
	createPVC(pvcClient, pvcName)
	watchPVC(pvcClient, pvcName)
	t := time.Now()
	elapsed := t.Sub(start)
	println("Setting time for: " + pvcName)
	timesPVC[i] = elapsed
}

func createPVC(pvcClient v2.PersistentVolumeClaimInterface, pvcName string) {

	var storage = resource.Quantity{}
	var storageClassName = *driver + "-csi"

	storage.Set(1024)
	pvc := core.PersistentVolumeClaim{
		TypeMeta: v1.TypeMeta{
			Kind: "PersitentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: pvcName,
		},
		Spec: core.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []core.PersistentVolumeAccessMode {
				"ReadWriteMany",
			},
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					"storage": storage,
				},
			},
		},
	}
	_, err := pvcClient.Create(context.Background(), &pvc, v1.CreateOptions{})
	if err != nil {
		fmt.Printf("%+v", err)
	}
}

func watchPVC(pvcClient v2.PersistentVolumeClaimInterface, pvcName string) {
	w, _ := pvcClient.Watch(context.Background(), v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", pvcName),
		Watch: true,
	})
	ev := w.ResultChan()

	for {
		select {
		case event := <- ev:
			switch event.Type {
			case watch.Modified:
				pvc := fmt.Sprintf("%s", event.Object)
				if strings.Contains(pvc, "Bound") {
					println(pvcName + ": " + "Bounded")
					return
				}
			}
		}
	}
}

func MeasurePodTimesThreaded(podClient v2.PodInterface, count int) {
	timesPod = make([]time.Duration, count)
	totalStartTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go Podworker(&wg, podClient, i)
	}

	wg.Wait()
	totalEndTime := time.Now()
	totalPodTime = totalEndTime.Sub(totalStartTime)
}

func Podworker(wg *sync.WaitGroup, podClient v2.PodInterface, i int) {
	defer wg.Done()
	var podName = podNamePattern + strconv.Itoa(i)
	var pvcName = pvcNamePattern + strconv.Itoa(i)
	createPod(podClient, podName, pvcName)
	start := time.Now()
	watchPod(podClient, podName)
	t := time.Now()
	elapsed := t.Sub(start)
	println("Setting time for: " + podName)
	timesPod[i] = elapsed
}

func createPod(podClient v2.PodInterface, podName string, pvcName string) {

	pod := core.Pod{
		TypeMeta: v1.TypeMeta{
			Kind: "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: podName,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name: podName,
					Image: "mcr.microsoft.com/oss/nginx/nginx:1.17.3-alpine",
					VolumeMounts: []core.VolumeMount{
						{
							Name: podName,
							MountPath: "/mnt/" + *driver,
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: podName,
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	_, err := podClient.Create(context.Background(), &pod, v1.CreateOptions{})
	if err != nil {
		fmt.Printf("%+v", err)
	}
}

func watchPod(podClient v2.PodInterface, podName string) {
	w, _ := podClient.Watch(context.Background(), v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
		Watch: true,
	})
	ev := w.ResultChan()

	for {
		select {
		case event := <- ev:
			switch event.Type {
			case watch.Modified:
				pod := fmt.Sprintf("%s", event.Object)
				if strings.Contains(pod, "Ready:true") {
					println(podName + ": " + "Running")
					return
				}
			}
		}
	}
}

func generateChart(data []time.Duration, title string, chartName string, msTolerated int64, msFailure int64) {
	var xAxis = make([]int, len(data))
	for i := 0 ; i < len(data); i++ {
		xAxis[i] = i
	}
	var average int64 = 0
	for i := 0 ; i < len(data); i++ {
		average += data[i].Milliseconds()
	}
	average /= int64(len(data))
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title:    title,
		Subtitle: fmt.Sprintf("Average time: %s ms", strconv.Itoa(int(average))),
	}))

	// Put data into instance
	bar.SetXAxis(xAxis).AddSeries("Category A", generateBarItems(data, msTolerated, msFailure))
	//AddSeries("Category B", generateBarItems())
	// Where the magic happens
	f, _ := os.Create(chartName)
	bar.Render(f)
}

func generateBarItems(data []time.Duration, msTolerated int64, msFailure int64) []opts.BarData {
	items := make([]opts.BarData, 0)
	for i := 0; i < len(data); i++ {
		items = append(items, opts.BarData{
			Value: data[i].Milliseconds(),
			ItemStyle: &opts.ItemStyle{
				Color: func(millis int64) string {
					if msTolerated == 0 && msFailure == 0 {
						return ""
					}
					if millis > msFailure {
						return "red"
					}
					if millis > msTolerated {
						return "yellow"
					}
					return "green"
				}(data[i].Milliseconds()),
			},
		})
	}
	return items
}
