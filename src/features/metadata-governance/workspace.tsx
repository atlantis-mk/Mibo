import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'
import {
  getMediaCardMatchStatus,
  getMediaCardMetadataProvider,
  getMediaCardPosterUrl,
} from '@/lib/media-presentation'
import type { HomeContentSection } from '@/lib/mibo-api'
import { createAuthedMiboApi, miboQueryKeys } from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getDisplayMatchStatus } from '@/features/media/components/standalone-media-detail-utils'
import { formatMatchStatus, formatMediaType } from './formatters'

const PAGE_SIZE = 10

export function MetadataGovernanceWorkspace({ token }: { token: string }) {
  const [page, setPage] = useState(1)
  const homeSectionsQuery = useQuery({
    queryKey: miboQueryKeys.metadataWorkspace(token),
    queryFn: () => createAuthedMiboApi(token).homeSections(),
  })
  const rows = flattenHomeSections(homeSectionsQuery.data)
  const totalPages = Math.max(1, Math.ceil(rows.length / PAGE_SIZE))
  const currentPage = Math.min(page, totalPages)
  const pageRows = rows.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE
  )
  const hasPreviousPage = currentPage > 1
  const hasNextPage = currentPage < totalPages

  useEffect(() => {
    setPage((current) => Math.min(current, totalPages))
  }, [totalPages])

  return (
    <div className='flex h-full min-h-0 flex-col gap-4 text-foreground'>
      {homeSectionsQuery.isLoading ? (
        <WorkspaceLoadingState />
      ) : homeSectionsQuery.error ? (
        <Alert>
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{homeSectionsQuery.error.message}</AlertDescription>
        </Alert>
      ) : rows.length ? (
        <div className='flex min-h-0 flex-1 flex-col gap-4'>
          <div className='flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
            <div className='text-sm text-muted-foreground'>
              共 {rows.length} 个条目
            </div>
            <div className='text-sm text-muted-foreground'>
              第 {currentPage} / {totalPages} 页 · 每页 {PAGE_SIZE} 条
            </div>
          </div>

          <div className='min-h-0 flex-1 overflow-hidden rounded-xl border border-border/60 bg-background/50'>
            <div className='h-full overflow-auto'>
              <Table>
                <TableHeader className='sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80'>
                  <TableRow>
                    <TableHead className='w-[320px]'>条目</TableHead>
                    <TableHead>分组</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>来源</TableHead>
                    <TableHead className='w-[120px] text-right'>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageRows.map(({ item, sectionTitle }) => {
                    const posterUrl = getMediaCardPosterUrl(item)
                    const provider = getMediaCardMetadataProvider(item)

                    return (
                      <TableRow key={`${sectionTitle}-${item.id}`}>
                        <TableCell className='min-w-72 whitespace-normal'>
                          <div className='flex items-start gap-3 py-1'>
                            <div className='h-16 w-12 shrink-0 overflow-hidden rounded-md bg-muted'>
                              {posterUrl ? (
                                <img
                                  src={posterUrl}
                                  alt={item.title}
                                  className='h-full w-full object-cover'
                                />
                              ) : null}
                            </div>
                            <div className='min-w-0'>
                              <div className='line-clamp-2 font-medium text-foreground'>
                                {item.title}
                              </div>
                              <div className='mt-1 text-xs text-muted-foreground'>
                                {item.year ?? '年份未知'}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell className='whitespace-normal text-muted-foreground'>
                          {sectionTitle}
                        </TableCell>
                        <TableCell>{formatMediaType(item.type)}</TableCell>
                        <TableCell>
                          <Badge
                            variant='outline'
                            className='border-border/60 bg-card/70 text-[11px]'
                          >
                            {formatMatchStatus(
                              getDisplayMatchStatus({
                                governance_status:
                                  getMediaCardMatchStatus(item),
                              })
                            )}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {provider ? (
                            <Badge variant='secondary' className='text-[11px]'>
                              {provider.toUpperCase()}
                            </Badge>
                          ) : (
                            <span className='text-sm text-muted-foreground'>
                              未匹配
                            </span>
                          )}
                        </TableCell>
                        <TableCell className='text-right'>
                          <Button asChild size='sm'>
                            <Link
                              to='/settings/metadata/$id'
                              params={{ id: String(item.id) }}
                            >
                              进入治理
                            </Link>
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          </div>

          <div className='mt-auto flex shrink-0 flex-col gap-3 border-t border-border/60 pt-4 sm:flex-row sm:items-center sm:justify-between'>
            <div className='text-sm text-muted-foreground'>
              当前显示 {pageRows.length} 条
            </div>
            <Pagination className='mx-0 w-auto justify-start sm:justify-end'>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    text='上一页'
                    href='#'
                    aria-disabled={!hasPreviousPage}
                    className={
                      hasPreviousPage
                        ? undefined
                        : 'pointer-events-none opacity-50'
                    }
                    onClick={(event) => {
                      event.preventDefault()
                      if (hasPreviousPage) setPage((current) => current - 1)
                    }}
                  />
                </PaginationItem>
                <PaginationItem>
                  <PaginationNext
                    text='下一页'
                    href='#'
                    aria-disabled={!hasNextPage}
                    className={
                      hasNextPage ? undefined : 'pointer-events-none opacity-50'
                    }
                    onClick={(event) => {
                      event.preventDefault()
                      if (hasNextPage) setPage((current) => current + 1)
                    }}
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        </div>
      ) : (
        <div className='flex flex-1 items-center justify-center rounded-xl border border-dashed border-border/70 px-4 py-10 text-center text-sm text-muted-foreground'>
          当前没有可展示的条目。
        </div>
      )}
    </div>
  )
}

function flattenHomeSections(sections: HomeContentSection[] | undefined) {
  if (!sections?.length) return []

  return sections.flatMap((section) =>
    section.items.map((item) => ({
      item,
      sectionKey: section.key,
      sectionTitle: section.title,
    }))
  )
}

function WorkspaceLoadingState() {
  return (
    <div className='flex flex-1 items-center gap-3 rounded-[1.25rem] border border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground'>
      <LoaderCircleIcon className='size-4 animate-spin' />
      正在加载最近条目
    </div>
  )
}
