import { ContentSection } from '../components/content-section'
import { ProfileForm } from './profile-form'

export function SettingsProfile() {
  return (
    <ContentSection
      title='个人资料'
      desc='这些信息会决定其他用户如何看到你的公开资料。'
    >
      <ProfileForm />
    </ContentSection>
  )
}
