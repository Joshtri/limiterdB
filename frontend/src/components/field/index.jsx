import {cn} from '@/lib/cn'

export function Field({label, hint, className, children}) {
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <div className="flex items-baseline justify-between">
        <span className="text-xs font-medium text-(--color-secondary)">{label}</span>
        {hint && <span className="text-xs text-(--color-muted)">{hint}</span>}
      </div>
      {children}
    </div>
  )
}
