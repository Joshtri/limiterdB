import {Button as BaseButton} from '@base-ui/react/button'
import {cn} from '@/lib/cn'

const variants = {
  primary: 'bg-brand-500 text-white not-data-disabled:hover:bg-brand-400 focus-visible:ring-2 focus-visible:ring-brand-500',
  secondary: 'bg-(--color-surface-3) text-(--color-fg) border border-(--color-border) not-data-disabled:hover:bg-(--color-border-strong)',
  danger: 'bg-danger text-white not-data-disabled:hover:opacity-90',
}

const sizes = {
  sm: 'h-7 px-2.5 text-xs rounded-sm gap-1.5',
  md: 'h-9 px-4 text-sm rounded-md gap-1.5',
}

export function Button({variant = 'secondary', size = 'md', className, children, ...props}) {
  return (
    <BaseButton
      className={cn(
        'inline-flex items-center justify-center font-medium transition-colors select-none',
        'focus-visible:outline-2 focus-visible:-outline-offset-1',
        'data-disabled:pointer-events-none data-disabled:opacity-40',
        'cursor-pointer',
        variants[variant],
        sizes[size],
        className
      )}
      {...props}
    >
      {children}
    </BaseButton>
  )
}
