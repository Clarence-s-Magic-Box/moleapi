/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { API, showError } from '../../helpers';
import { marked } from 'marked';
import { Empty } from '@douyinfe/semi-ui';
import { IllustrationConstruction, IllustrationConstructionDark } from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';

const About = () => {
  const { t } = useTranslation();
  const [about, setAbout] = useState('');
  const [aboutLoaded, setAboutLoaded] = useState(false);
  const currentYear = new Date().getFullYear();

  const displayAbout = async () => {
    setAbout(localStorage.getItem('about') || '');
    const res = await API.get('/api/about');
    const { success, message, data } = res.data;
    if (success) {
      let aboutContent = data;
      if (!data.startsWith('https://')) {
        aboutContent = marked.parse(data);
      }
      setAbout(aboutContent);
      localStorage.setItem('about', aboutContent);
    } else {
      showError(message);
      setAbout(t('加载关于内容失败...'));
    }
    setAboutLoaded(true);
  };

  useEffect(() => {
    displayAbout().then();
  }, []);

  const emptyStyle = {
    padding: '24px',
  };

  const customDescription = (
    <div style={{ textAlign: 'center', fontSize: '14px', lineHeight: 1.7 }}>
      <p style={{ marginBottom: 8 }}>
        {t('关于项目')}: {' '}
        <a
          href='https://github.com/QuantumNous/new-api'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          QuantumNous/new-api
        </a>
      </p>

      <p style={{ marginBottom: 8 }}>
        <a
          href='https://github.com/QuantumNous/new-api'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          NewAPI
        </a>{' '}
        {t('© {{currentYear}}', { currentYear })}{' '}
        <a
          href='https://github.com/QuantumNous'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          QuantumNous
        </a>{' '}
        | {t('关于我们')}: {' '}
        <a
          href='https://github.com/QuantumNous'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          github.com/QuantumNous
        </a>
      </p>

      <p style={{ marginBottom: 8 }}>
        {t('| 基于')}{' '}
        <a
          href='https://github.com/songquanpeng/one-api/releases/tag/v0.5.4'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          One API v0.5.4
        </a>{' '}
        © 2023{' '}
        <a
          href='https://github.com/songquanpeng'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          JustSong
        </a>
      </p>

      <p style={{ marginBottom: 8 }}>
        {t('协议与许可')}: {' '}
        <a
          href='https://www.gnu.org/licenses/agpl-3.0.html'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          AGPL-3.0
        </a>{' '}
        | <a
          href='https://github.com/QuantumNous/new-api/blob/main/LICENSE'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          LICENSE
        </a>{' '}
        | {t('上游 One API 许可证')}: {' '}
        <a
          href='https://github.com/songquanpeng/one-api/blob/v0.5.4/LICENSE'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          MIT
        </a>
      </p>

      <p style={{ marginTop: 12, marginBottom: 0, opacity: 0.85 }}>
        {t('本站仅提供 API 网关/转发与管理能力，不提供任何大模型服务本体；请在使用时遵守上游服务条款与当地法律法规。')}
      </p>
    </div>
  );

  return (
    <div className='mt-[60px] px-2'>
      {aboutLoaded && about === '' ? (
        <div className='flex justify-center items-center h-screen p-8'>
          <Empty
            image={<IllustrationConstruction style={{ width: 150, height: 150 }} />}
            darkModeImage={<IllustrationConstructionDark style={{ width: 150, height: 150 }} />}
            description={t('关于项目')}
            style={emptyStyle}
          >
            {customDescription}
          </Empty>
        </div>
      ) : (
        <>
          {about.startsWith('https://') ? (
            <iframe
              src={about}
              style={{ width: '100%', height: '100vh', border: 'none' }}
            />
          ) : (
            <div
              style={{ fontSize: 'larger' }}
              dangerouslySetInnerHTML={{ __html: about }}
            ></div>
          )}
        </>
      )}
    </div>
  );
};

export default About;
