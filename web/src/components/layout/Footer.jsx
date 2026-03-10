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

import React, { useEffect, useState, useMemo, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { Typography } from '@douyinfe/semi-ui';
import { getFooterHTML, getSystemName } from '../../helpers';
import { StatusContext } from '../../context/Status';

const FooterBar = () => {
  const { t, i18n } = useTranslation();
  const [footer, setFooter] = useState(getFooterHTML());
  const systemName = getSystemName();
  const [statusState] = useContext(StatusContext);
  const upstreamVersion =
    statusState?.status?.upstream_version || statusState?.status?.version || '';
  const projectVersion = statusState?.status?.project_version || '';
  const brandingHint = i18n.language.startsWith('zh')
    ? `${systemName} 是 MoleAPI 的大模型控制台，强调高速、稳定与真实官方供应商接入。`
    : `${systemName} is the MoleAPI control console focused on fast, stable access to official model providers.`;

  const loadFooter = () => {
    let footer_html = localStorage.getItem('footer_html');
    if (footer_html) {
      setFooter(footer_html);
    }
  };

  const currentYear = new Date().getFullYear();

  const customFooter = useMemo(() => {
    return (
      <footer
        className='relative h-auto py-8 px-6 md:px-24 w-full flex flex-col items-center justify-between overflow-hidden mt-6 border-none'
        style={{ border: 'none', outline: 'none' }}
      >
        <div className='flex flex-col items-center justify-center w-full max-w-[1110px] gap-1 text-center'>
          <div className='text-xs flex flex-wrap items-center justify-center gap-2'>
            <Typography.Text className='!text-semi-color-text-1'>
              © {currentYear} {systemName}. {t('设计与开发由')}{' '}
              <a
                href='https://github.com/ClarenceDan'
                target='_blank'
                rel='noreferrer'
                className='!text-semi-color-primary font-medium'
              >
                ClarenceDan
              </a>
            </Typography.Text>
          </div>

          <div className='text-xs !text-semi-color-text-1'>
            <div className='footer-sub mb-1'>{brandingHint}</div>
            <div className='footer-sub'>
              Based on{' '}
              <a
                href='https://github.com/QuantumNous/new-api'
                target='_blank'
                rel='noreferrer'
                className='!text-semi-color-primary font-medium'
              >
                New API {upstreamVersion}
              </a>{' '}
              {projectVersion ? (
                <>
                  {' '}
                  || MoleAPI {projectVersion}
                </>
              ) : null}{' '}
              by{' '}
              <a
                href='https://github.com/QuantumNous'
                target='_blank'
                rel='noreferrer'
                className='!text-semi-color-primary font-medium'
              >
                QuantumNous
              </a>{' '}
              ||{' '}
              <a
                href='https://github.com/songquanpeng/one-api'
                target='_blank'
                rel='noreferrer'
                className='!text-semi-color-primary font-medium'
              >
                One API
              </a>{' '}
              by{' '}
              <a
                href='https://github.com/songquanpeng'
                target='_blank'
                rel='noreferrer'
                className='!text-semi-color-primary font-medium'
              >
                JustSong
              </a>
            </div>
          </div>
        </div>
      </footer>
    );
  }, [
    brandingHint,
    currentYear,
    i18n.language,
    projectVersion,
    systemName,
    t,
    upstreamVersion,
  ]);

  useEffect(() => {
    loadFooter();
  }, []);

  return (
    <div className='w-full'>
      {footer ? (
        <div className='relative'>
          <div
            className='custom-footer'
            dangerouslySetInnerHTML={{ __html: footer }}
          ></div>
        </div>
      ) : (
        customFooter
      )}
    </div>
  );
};

export default FooterBar;
