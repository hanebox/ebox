import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import type { Project } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/dialog'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Keep the name under 255 characters'),
  description: z.string().max(2000, 'Keep the description under 2000 characters').optional(),
})

type FormValues = z.infer<typeof schema>

export interface ProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  project?: Project
  onSaved?: (project: Project) => void
}

export function ProjectDialog({ open, onOpenChange, project, onSaved }: ProjectDialogProps) {
  const queryClient = useQueryClient()
  const editing = Boolean(project)

  const defaultValues = useMemo<FormValues>(
    () => ({ name: project?.name ?? '', description: project?.description ?? '' }),
    [project],
  )

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: defaultValues,
  })

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      project ? api.updateProject(project.uuid, values) : api.createProject(values),
    onSuccess: (saved) => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] })
      if (project) {
        void queryClient.invalidateQueries({ queryKey: ['project', project.uuid] })
      }
      toast.success(editing ? 'Project updated' : 'Project created', saved.name)
      onOpenChange(false)
      form.reset()
      onSaved?.(saved)
    },
    onError: (error) => {
      toast.error(
        editing ? 'Could not update the project' : 'Could not create the project',
        error instanceof ApiError ? error.message : undefined,
      )
    },
  })

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={editing ? 'Edit project' : 'New project'}
      description={
        editing
          ? 'Rename the project or update its description.'
          : 'Projects group the applications, databases and services of one environment.'
      }
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <Field label="Name" error={form.formState.errors.name?.message}>
          {(field) => (
            <Input
              {...field}
              {...form.register('name')}
              placeholder="personal"
              autoFocus
              invalid={Boolean(form.formState.errors.name)}
            />
          )}
        </Field>

        <Field
          label="Description"
          description="Optional. A short note about what lives in this project."
          error={form.formState.errors.description?.message}
        >
          {(field) => (
            <Textarea
              {...field}
              {...form.register('description')}
              rows={3}
              placeholder="My personal services"
              invalid={Boolean(form.formState.errors.description)}
            />
          )}
        </Field>

        <div className="flex items-center justify-end gap-2 pt-1">
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={mutation.isPending}>
            {editing ? 'Save changes' : 'Create project'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
