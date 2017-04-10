//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package virtcontainers

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Process gathers data related to a container process.
type Process struct {
	Token string
	Pid   int
}

// ContainerStatus describes a container status.
type ContainerStatus struct {
	ID     string
	State  State
	PID    int
	RootFs string
}

// ContainerConfig describes one container runtime configuration.
type ContainerConfig struct {
	ID string

	// RootFs is the container workload image on the host.
	RootFs string

	// Interactive specifies if the container runs in the foreground.
	Interactive bool

	// Console is a console path provided by the caller.
	Console string

	// Cmd specifies the command to run on a container
	Cmd Cmd
}

// valid checks that the container configuration is valid.
func (containerConfig *ContainerConfig) valid() bool {
	if containerConfig == nil {
		return false
	}

	if containerConfig.ID == "" {
		return false
	}

	return true
}

// Container is composed of a set of containers and a runtime environment.
// A Container can be created, deleted, started, stopped, listed, entered, paused and restored.
type Container struct {
	id    string
	podID string

	rootFs string

	config *ContainerConfig

	pod *Pod

	runPath       string
	configPath    string
	containerPath string

	state State

	process Process
}

// ID returns the container identifier string.
func (c *Container) ID() string {
	return c.id
}

// Process returns the container process.
func (c *Container) Process() Process {
	return c.process
}

// GetToken returns the token related to this container's process.
func (c *Container) GetToken() string {
	return c.process.Token
}

// GetPid returns the pid related to this container's process.
func (c *Container) GetPid() int {
	return c.process.Pid
}

// SetPid sets and stores the given pid as the pid of container's process.
func (c *Container) SetPid(pid int) error {
	c.process.Pid = pid

	return c.storeProcess()
}

func (c *Container) storeProcess() error {
	return c.pod.storage.storeContainerProcess(c.podID, c.id, c.process)
}

func (c *Container) fetchProcess() (Process, error) {
	return c.pod.storage.fetchContainerProcess(c.podID, c.id)
}

// fetchContainer fetches a container config from a pod ID and returns a Container.
func fetchContainer(pod *Pod, containerID string) (*Container, error) {
	if pod == nil {
		return nil, ErrNeedPod
	}

	if containerID == "" {
		return nil, ErrNeedContainerID
	}

	fs := filesystem{}
	config, err := fs.fetchContainerConfig(pod.id, containerID)
	if err != nil {
		return nil, err
	}

	virtLog.Infof("Info structure: %+v", config)

	return createContainer(pod, config)
}

// storeContainer stores a container config.
func (c *Container) storeContainer() error {
	fs := filesystem{}
	err := fs.storeContainerResource(c.pod.id, c.id, configFileType, *(c.config))
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) setContainerState(state stateString) error {
	if state == "" {
		return ErrNeedState
	}

	c.state = State{
		State: state,
	}

	err := c.pod.storage.storeContainerResource(c.podID, c.id, stateFileType, c.state)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) createContainersDirs() error {
	err := os.MkdirAll(c.runPath, dirMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(c.configPath, dirMode)
	if err != nil {
		c.pod.storage.deleteContainerResources(c.podID, c.id, nil)
		return err
	}

	return nil
}

func createContainers(pod *Pod, contConfigs []ContainerConfig) ([]*Container, error) {
	if pod == nil {
		return nil, ErrNeedPod
	}

	var containers []*Container

	for _, contConfig := range contConfigs {
		if contConfig.valid() == false {
			return containers, fmt.Errorf("Invalid container configuration")
		}

		c := &Container{
			id:            contConfig.ID,
			podID:         pod.id,
			rootFs:        contConfig.RootFs,
			config:        &contConfig,
			pod:           pod,
			runPath:       filepath.Join(runStoragePath, pod.id, contConfig.ID),
			configPath:    filepath.Join(configStoragePath, pod.id, contConfig.ID),
			containerPath: filepath.Join(pod.id, contConfig.ID),
			state:         State{},
			process:       Process{},
		}

		state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
		if err == nil {
			c.state.State = state.State
		}

		process, err := c.pod.storage.fetchContainerProcess(c.podID, c.id)
		if err == nil {
			c.process = process
		}

		containers = append(containers, c)
	}

	return containers, nil
}

func createContainer(pod *Pod, contConfig ContainerConfig) (*Container, error) {
	fmt.Printf("DEBUG: createContainer: pod=%v, contConfig=%v\n", pod, contConfig)
	if pod == nil {
		return nil, ErrNeedPod
	}

	if contConfig.valid() == false {
		return nil, fmt.Errorf("Invalid container configuration")
	}

	c := &Container{
		id:            contConfig.ID,
		podID:         pod.id,
		rootFs:        contConfig.RootFs,
		config:        &contConfig,
		pod:           pod,
		runPath:       filepath.Join(runStoragePath, pod.id, contConfig.ID),
		configPath:    filepath.Join(configStoragePath, pod.id, contConfig.ID),
		containerPath: filepath.Join(pod.id, contConfig.ID),
		state:         State{},
		process:       Process{},
	}

	err := c.createContainersDirs()
	fmt.Printf("DEBUG: createContainer: c.createContainersDirs: err=%v\n", err)
	if err != nil {
		return nil, err
	}

	process, err := c.fetchProcess()
	fmt.Printf("DEBUG: createContainer: c.fetchProcess: process=%v, err=%v\n", process, err)
	if err == nil {
		c.process = process
	}

	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	fmt.Printf("DEBUG: createContainer: c.pod.storage.fetchContainerState(c.podID=%v, c.id=%v) state=%v, err=%v\n", c.podID, c.id, state, err)
	if err == nil && state.State != "" {
		c.state.State = state.State
		return c, nil
	}

	// If we reached that point, this means that no state file has been
	// found and that we are in the first creation of this container.
	// We don't want the following code to be executed outside of this
	// specific case.
	pod.containers = append(pod.containers, c)

	if err := c.pod.agent.createContainer(pod, c); err != nil {
		fmt.Printf("DEBUG: createContainer: c.pod.agent.createContainer err=%v\n", err)
		return nil, err
	}

	if err := c.pod.setContainerState(c.id, StateReady); err != nil {
		fmt.Printf("DEBUG: createContainer: c.pod.setContainerState(c.id=%v, StateReady), err=%v\n", c.id, err)
		return nil, err
	}

	return c, nil
}

func (c *Container) delete() error {
	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	fmt.Printf("DEBUG: createContainer: c.pod.storage.fetchContainerState(c.podID=%v, c.id=%v) state=%v, err=%v\n", c.podID, c.id, state, err)

	if err != nil {
		return err
	}

	if state.State != StateReady && state.State != StateStopped {
		return fmt.Errorf("Container not ready or stopped, impossible to delete")
	}

	err = c.pod.storage.deleteContainerResources(c.podID, c.id, nil)
	fmt.Printf("DEBUG: createContainer: c.pod.storage.deleteContainerResources err=%v\n", err)
	if err != nil {
		return err
	}

	return nil
}

// fetchState retrieves the container state.
//
// cmd specifies the operation (or verb) that the retieval is destined
// for and is only used to make the returned error as descriptive as
// possible.
func (c *Container) fetchState(cmd string) (State, error) {
	fmt.Printf("DEBUG: fetchState: cmd=%v\n", cmd)
	if cmd == "" {
		return State{}, fmt.Errorf("Cmd cannot be empty")
	}

	state, err := c.pod.storage.fetchPodState(c.pod.id)
	fmt.Printf("DEBUG: fetchState: c.pod.storage.fetchPodState(c.pod.id=%v) state=%v, err=%v\n", c.pod.id, state, err)
	if err != nil {
		return State{}, err
	}

	if state.State != StateRunning {
		return State{}, fmt.Errorf("Pod not running, impossible to %s the container", cmd)
	}

	state, err = c.pod.storage.fetchContainerState(c.podID, c.id)
	fmt.Printf("DEBUG: fetchState: c.pod.storage.fetchContainerState(c.pod.id=%v, c.id=%v) state=%v, err=%v\n", c.pod.id, c.id, state, err)
	if err != nil {
		return State{}, err
	}

	return state, nil
}

func (c *Container) start() error {
	state, err := c.fetchState("start")
	fmt.Printf("DEBUG: Container.start: c.fetchState('start'): state=%v, err=%v\n", state, err)
	if err != nil {
		return err
	}

	if state.State != StateReady && state.State != StateStopped {
		return fmt.Errorf("Container not ready or stopped, impossible to start")
	}

	err = state.validTransition(StateReady, StateRunning)
	fmt.Printf("DEBUG: Container.start: state.validTransition(StateReady, StateRunning): err=%v\n", err)
	if err != nil {
		err = state.validTransition(StateStopped, StateRunning)
		fmt.Printf("DEBUG: Container.start: state.validTransition(StateStopped, StateRunning): err=%v\n", err)
		if err != nil {
			return err
		}
	}

	err = c.pod.agent.startContainer(*(c.pod), *c)
	fmt.Printf("DEBUG: Container.start: c.pod.agent.startContainer err=%v\n", err)
	if err != nil {
		c.stop()
		return err
	}

	err = c.setContainerState(StateRunning)
	fmt.Printf("DEBUG: Container.start: c.setContainerState(StateRunning) err=%v\n", err)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) stop() error {
	state, err := c.fetchState("stop")
	fmt.Printf("DEBUG: Container.stop: c.fetchState('stop') state=%v, err=%v\n", state, err)
	if err != nil {
		return err
	}

	if state.State != StateRunning {
		return fmt.Errorf("Container not running, impossible to stop")
	}

	err = state.validTransition(StateRunning, StateStopped)
	fmt.Printf("DEBUG: Container.stop: state.validTransition(StateRunning, StateStopped) err=%v\n", err)
	if err != nil {
		return err
	}

	err = c.pod.agent.killContainer(*(c.pod), *c, syscall.SIGTERM)
	fmt.Printf("DEBUG: Container.stop: c.pod.agent.killContainer err=%v\n", err)
	if err != nil {
		return err
	}

	err = c.pod.agent.stopContainer(*(c.pod), *c)
	fmt.Printf("DEBUG: Container.stop: c.pod.agent.stopContainer err=%v\n", err)
	if err != nil {
		return err
	}

	err = c.setContainerState(StateStopped)
	fmt.Printf("DEBUG: Container.stop: c.setContainerState(StateStopped) err=%v\n", err)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) enter(cmd Cmd) (*Process, error) {
	state, err := c.fetchState("enter")
	if err != nil {
		return nil, err
	}

	if state.State != StateRunning {
		return nil, fmt.Errorf("Container not running, impossible to enter")
	}

	process, err := c.pod.agent.exec(c.pod, *c, cmd)
	if err != nil {
		return nil, err
	}

	return process, nil
}

func (c *Container) kill(signal syscall.Signal) error {
	state, err := c.fetchState("signal")
	fmt.Printf("DEBUG: Container.kill: c.fetchState('signal') state=%v, err=%v\n", state, err)
	if err != nil {
		return err
	}

	if state.State != StateRunning {
		return fmt.Errorf("Container not running, impossible to signal the container")
	}

	err = c.pod.agent.killContainer(*(c.pod), *c, signal)
	fmt.Printf("DEBUG: Container.kill: c.pod.agent.killContainer err=%v\n", err)
	if err != nil {
		return err
	}

	return nil
}
