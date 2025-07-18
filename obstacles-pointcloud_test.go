package obstaclespointcloud

import (
	"context"
	"image/color"
	"testing"

	"github.com/pkg/errors"
	"go.viam.com/test"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	pc "go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
	"go.viam.com/rdk/testutils/inject"
)

func TestRadiusClusteringSegmentation(t *testing.T) {
	r := &inject.Robot{}
	r.LoggerFunc = func() logging.Logger { return nil }
	cam := &inject.Camera{}
	cam.NextPointCloudFunc = func(ctx context.Context) (pc.PointCloud, error) {
		return nil, errors.New("no pointcloud")
	}
	r.ResourceNamesFunc = func() []resource.Name {
		return []resource.Name{camera.Named("fakeCamera")}
	}
	r.ResourceByNameFunc = func(n resource.Name) (resource.Resource, error) {
		switch n.Name {
		case "fakeCamera":
			return cam, nil
		default:
			return nil, resource.NewNotFoundError(n)
		}
	}

	// Create dependencies map
	deps := make(resource.Dependencies)
	deps[camera.Named("fakeCamera")] = cam

	params := &ObstaclesPointCloudConfig{
		MinPtsInPlane:        100,
		MaxDistFromPlane:     10,
		MinPtsInSegment:      3,
		AngleTolerance:       20,
		ClusteringRadius:     5,
		ClusteringStrictness: 3,
		DefaultCamera:        "fakeCamera",
	}
	// bad registration, no parameters
	name := vision.Named("test_rcs")
	_, err := registerPointCloudSegmenter(context.Background(), name, nil, deps, nil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "cannot be nil")
	// bad registration, parameters out of bounds
	params.ClusteringRadius = -3
	_, err = registerPointCloudSegmenter(context.Background(), name, params, deps, nil)
	test.That(t, err, test.ShouldBeNil)
	// successful registration
	params.ClusteringRadius = 1
	seg, err := registerPointCloudSegmenter(context.Background(), name, params, deps, nil)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, seg.Name(), test.ShouldResemble, name)
	// successful registration, valid default camera
	params.DefaultCamera = "fakeCamera"
	_, err = registerPointCloudSegmenter(context.Background(), name, params, deps, nil)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, seg.Name(), test.ShouldResemble, name)
	// successful registration, invalid default camera
	params.DefaultCamera = "not-camera"
	_, err = registerPointCloudSegmenter(context.Background(), name, params, deps, nil)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "could not find camera \"not-camera\"")
	// Test properties. Should support object PCDs and not detections or classifications
	params.DefaultCamera = "fakeCamera"
	seg, _ = registerPointCloudSegmenter(context.Background(), name, params, deps, nil)
	props, err := seg.GetProperties(context.Background(), nil)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, props.ObjectPCDsSupported, test.ShouldEqual, true)
	test.That(t, props.DetectionSupported, test.ShouldEqual, false)
	test.That(t, props.ClassificationSupported, test.ShouldEqual, false)
	// fails on not finding camera
	_, err = seg.GetObjectPointClouds(context.Background(), "no_camera", map[string]interface{}{})
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "Resource missing from dependencies")
	// fails since camera cannot generate point clouds
	_, err = seg.GetObjectPointClouds(context.Background(), "fakeCamera", map[string]interface{}{})
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "no pointcloud")
	// successful, creates two clusters of points
	cam.NextPointCloudFunc = func(ctx context.Context) (pc.PointCloud, error) {
		cloud := pc.NewBasicEmpty()
		// cluster 1
		err = cloud.Set(pc.NewVector(1, 1, 1), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		err = cloud.Set(pc.NewVector(1, 1, 2), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		err = cloud.Set(pc.NewVector(1, 1, 3), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		err = cloud.Set(pc.NewVector(1, 1, 4), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		// cluster 2
		err = cloud.Set(pc.NewVector(2, 2, 101), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		err = cloud.Set(pc.NewVector(2, 2, 102), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		err = cloud.Set(pc.NewVector(2, 2, 103), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		err = cloud.Set(pc.NewVector(2, 2, 104), pc.NewColoredData(color.NRGBA{255, 0, 0, 255}))
		test.That(t, err, test.ShouldBeNil)
		return cloud, nil
	}
	// objects, err := seg.GetObjectPointClouds(context.Background(), "fakeCamera", nil)
	// test.That(t, err, test.ShouldBeNil)
	// test.That(t, len(objects), test.ShouldEqual, 2)
	// does not implement detector
	_, err = seg.Detections(context.Background(), nil, nil)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "does not implement")
}
