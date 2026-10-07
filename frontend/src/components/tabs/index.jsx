import {Tabs as BaseTabs} from '@base-ui/react/tabs'
import {cn} from '@/lib/cn'

function Root({className, ...props}) {
  return <BaseTabs.Root className={cn('flex flex-col', className)} {...props} />
}

function List({className, children, ...props}) {
  return (
    <BaseTabs.List
      className={cn('relative z-0 flex gap-0.5 rounded-md bg-(--color-surface-2) p-0.5', className)}
      {...props}
    >
      {children}
      <BaseTabs.Indicator className="absolute top-0.5 left-0 -z-1 h-[calc(100%-4px)] w-(--active-tab-width) translate-x-(--active-tab-left) rounded-sm bg-(--color-surface-3) transition-[translate,width] duration-150 ease-in-out" />
    </BaseTabs.List>
  )
}

function Tab({className, ...props}) {
  return (
    <BaseTabs.Tab
      className={cn(
        'h-7 px-3 rounded-sm text-xs font-medium text-(--color-muted) select-none cursor-pointer',
        'hover:text-(--color-fg) data-active:text-(--color-fg)',
        'focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-brand-500',
        className
      )}
      {...props}
    />
  )
}

function Panel({className, ...props}) {
  return <BaseTabs.Panel className={cn('outline-none [[hidden]]:hidden', className)} {...props} />
}

export const Tabs = {Root, List, Tab, Panel}
