import { ContentSection } from '../components/content-section'
import { DisplayForm } from './display-form'

export function SettingsDisplay() {
  return (
    <ContentSection
      title='播放'
      desc='控制自动连播、直放偏好、默认字幕策略，以及内置/外部播放器打开方式。'
    >
      <DisplayForm />
    </ContentSection>
  )
}
