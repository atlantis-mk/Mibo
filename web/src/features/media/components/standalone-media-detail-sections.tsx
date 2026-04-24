import { ChevronLeft, ChevronRight } from 'lucide-react'
import { FreeMode } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperType } from 'swiper/types'

import { Button } from '#/components/ui/button'
import type { MediaItemDetail } from '#/lib/mibo-api'

export { DetailHeroSection } from './standalone-media-detail-hero'
export { SpecsSection } from './standalone-media-detail-specs'

export function CastSection({
  item,
  canScrollCastPrev,
  canScrollCastNext,
  onSwiper,
  onSlideChange,
  onPrev,
  onNext,
}: {
  item: MediaItemDetail
  canScrollCastPrev: boolean
  canScrollCastNext: boolean
  onSwiper: (swiper: SwiperType) => void
  onSlideChange: (swiper: SwiperType) => void
  onPrev: () => void
  onNext: () => void
}) {
  if ((item.cast ?? []).length === 0) return null

  return (
    <section className="mt-10 space-y-4">
      <h2 className="text-[19px] font-semibold text-foreground">演职人员</h2>
      <div className="relative px-12 sm:px-14">
        <Swiper
          modules={[FreeMode]}
          freeMode
          slidesPerView="auto"
          spaceBetween={20}
          onSwiper={onSwiper}
          onSlideChange={onSlideChange}
          onResize={onSlideChange}
          className="w-full"
        >
          {(item.cast ?? []).map((person) => (
            <SwiperSlide
              key={`${person.name}-${person.role}`}
              className="!h-auto !w-[190px] sm:!w-[210px]"
            >
              <div className="overflow-hidden rounded-[8px] border border-border/40 bg-card/70 shadow-lg">
                <div className="relative aspect-[4/5] bg-[linear-gradient(160deg,rgba(255,255,255,0.12),rgba(255,255,255,0.02)),linear-gradient(180deg,rgba(24,24,27,0.25),rgba(24,24,27,0.9))]">
                  {person.avatar_url ? (
                    <img
                      src={person.avatar_url}
                      alt={person.name}
                      className="absolute inset-0 h-full w-full object-cover"
                    />
                  ) : null}
                </div>
              </div>
              <div className="px-1 pt-3 text-center">
                <div className="line-clamp-1 text-lg text-foreground">
                  {person.name}
                </div>
                <div className="line-clamp-2 text-sm leading-6 text-muted-foreground">
                  {person.role || item.title}
                </div>
              </div>
            </SwiperSlide>
          ))}
        </Swiper>
        <Button
          type="button"
          size="icon-sm"
          variant="outline"
          className="absolute left-0 top-1/2 -translate-y-1/2 rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
          onClick={onPrev}
          disabled={!canScrollCastPrev}
        >
          <ChevronLeft className="size-4" />
          <span className="sr-only">上一组演职人员</span>
        </Button>
        <Button
          type="button"
          size="icon-sm"
          variant="outline"
          className="absolute right-0 top-1/2 -translate-y-1/2 rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
          onClick={onNext}
          disabled={!canScrollCastNext}
        >
          <ChevronRight className="size-4" />
          <span className="sr-only">下一组演职人员</span>
        </Button>
      </div>
    </section>
  )
}
