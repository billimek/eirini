package k8s

import (
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/opi"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
)

const ActiveDeadlineSeconds = 900

type TaskDesirer struct {
	Namespace       string
	CCUploaderIP    string
	CertsSecretName string
	Client          kubernetes.Interface
}

func (d *TaskDesirer) Desire(task *opi.Task) error {
	_, err := d.Client.BatchV1().Jobs(d.Namespace).Create(toJob(task))
	return err
}

func (d *TaskDesirer) DesireStaging(task *opi.Task) error {
	job := d.toStagingJob(task)
	_, err := d.Client.BatchV1().Jobs(d.Namespace).Create(job)
	return err
}

func (d *TaskDesirer) Delete(name string) error {
	backgroundPropagation := meta_v1.DeletePropagationBackground
	return d.Client.BatchV1().Jobs(d.Namespace).Delete(name, &meta_v1.DeleteOptions{
		PropagationPolicy: &backgroundPropagation,
	})
}

func (d *TaskDesirer) toStagingJob(task *opi.Task) *batch.Job {
	job := toJob(task)
	job.Spec.Template.Spec.HostAliases = []v1.HostAlias{
		{
			IP:        d.CCUploaderIP,
			Hostnames: []string{eirini.CCUploaderInternalURL},
		},
	}
	job.Spec.Template.Spec.Volumes = []v1.Volume{
		{
			Name: eirini.CCCertsVolumeName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: d.CertsSecretName,
				},
			},
		},
	}

	job.Spec.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
		{
			Name:      eirini.CCCertsVolumeName,
			ReadOnly:  true,
			MountPath: eirini.CCCertsMountPath,
		},
	}
	return job
}

func toJob(task *opi.Task) *batch.Job {
	job := &batch.Job{
		Spec: batch.JobSpec{
			ActiveDeadlineSeconds: int64ptr(ActiveDeadlineSeconds),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  "opi-task",
						Image: task.Image,
						Env:   MapToEnvVar(task.Env),
					}},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
		},
	}

	job.Name = task.Env[eirini.EnvStagingGUID]

	job.Spec.Template.Labels = map[string]string{
		"name": task.Env[eirini.EnvAppID],
	}

	job.Labels = map[string]string{
		"name": task.Env[eirini.EnvAppID],
	}
	return job
}
