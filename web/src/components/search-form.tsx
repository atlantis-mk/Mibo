'use client'

import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'

import { Label } from '#/components/ui/label'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarInput,
} from '#/components/ui/sidebar'
import { SearchIcon } from 'lucide-react'

export function SearchForm({ ...props }: React.ComponentProps<'form'>) {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')

  return (
    <form
      {...props}
      onSubmit={(event) => {
        event.preventDefault()
        void navigate({
          to: '/search',
          search: { q: query.trim() || undefined },
        })
      }}
    >
      <SidebarGroup className="py-0">
        <SidebarGroupContent className="relative">
          <Label htmlFor="search" className="sr-only">
            Search Mibo
          </Label>
          <SidebarInput
            id="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="搜索媒体库"
            className="pl-8"
          />
          <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-4 -translate-y-1/2 opacity-50 select-none" />
        </SidebarGroupContent>
      </SidebarGroup>
    </form>
  )
}
