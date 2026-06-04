import { ContentSection } from '../components/content-section'
import { NotificationsForm } from './notifications-form'

export function SettingsNotifications() {
  return (
    <ContentSection title='通知' desc='配置你接收站内通知和邮件通知的方式。'>
      <NotificationsForm />
    </ContentSection>
  )
}
