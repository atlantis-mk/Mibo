import type { OperationsTask } from '@/lib/mibo-api'

export function operationsSeverityLabel(severity: OperationsTask['severity']) {
  switch (severity) {
    case 'blocking':
      return '阻断'
    case 'error':
      return '错误'
    case 'warning':
      return '警告'
    case 'info':
      return '提示'
  }
}

export function operationsSeverityClassName(
  severity: OperationsTask['severity']
) {
  switch (severity) {
    case 'blocking':
      return 'border-destructive/30 bg-destructive/10 text-destructive'
    case 'error':
      return 'border-orange-500/30 bg-orange-500/10 text-orange-700'
    case 'warning':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-700'
    case 'info':
      return 'border-sky-500/30 bg-sky-500/10 text-sky-700'
  }
}

export function operationTaskTitle(task: OperationsTask) {
  return task.title
}

export function operationTaskMessage(task: OperationsTask) {
  return task.summary
}

export function findBlockingHomeTask(tasks: OperationsTask[]) {
  return tasks.find(
    (task) => task.severity === 'blocking' && task.impact.blocks_home_visibility
  )
}

export function affectedLibraryNames(task: OperationsTask) {
  return task.affected.libraries.map((library) => library.name).join('、')
}
