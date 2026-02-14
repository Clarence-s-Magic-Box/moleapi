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

import React, { useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Typography } from '@douyinfe/semi-ui';
import { getFooterHTML, getLogo, getSystemName } from '../../helpers';

const FooterBar = () => {
  const { t } = useTranslation();
  const [footer, setFooter] = useState(getFooterHTML());
  const systemName = getSystemName();
  const logo = getLogo();


  const loadFooter = () => {
    let footer_html = localStorage.getItem('footer_html');
    if (footer_html) {
      setFooter(footer_html);
    }
  };

  const currentYear = new Date().getFullYear();

  const customFooter = useMemo(() => (
    <footer className="relative h-auto py-8 px-6 md:px-24 w-full flex flex-col items-center justify-between overflow-hidden mt-6 border-none" style={{ border: 'none', outline: 'none' }}>



      <div className="flex flex-col items-center justify-center w-full max-w-[1110px] gap-1 text-center">
        <div className="text-xs flex flex-wrap items-center justify-center gap-2">
          <Typography.Text className="!text-semi-color-text-1">© {currentYear} {systemName}.Made with ❤️ by{' '}
            <a href="https://github.com/ClarenceDan" target="_blank" rel="noreferrer" className="!text-semi-color-primary font-medium">
              ClarenceDan
            </a> </Typography.Text>
        </div>

        <div className="text-xs !text-semi-color-text-1">
          <div className="footer-sub">
            Based on{' '}
            <a href="https://github.com/Calcium-Ion/new-api" target="_blank" rel="noreferrer" className="!text-semi-color-primary font-medium">
              New API {import.meta.env.VITE_REACT_APP_VERSION}
            </a>{' '}
            by{' '}
            <a href="https://github.com/Calcium-Ion" target="_blank" rel="noreferrer" className="!text-semi-color-primary font-medium">
              Calcium-Ion
            </a>{' '}
            ||{' '}
            <a href="https://github.com/songquanpeng/one-api" target="_blank" rel="noreferrer" className="!text-semi-color-primary font-medium">
              One API
            </a>{' '}
            by{' '}
            <a href="https://github.com/songquanpeng" target="_blank" rel="noreferrer" className="!text-semi-color-primary font-medium">
              JustSong
            </a>
          </div>
        </div>
      </div>
    </footer>
  ), [logo, systemName, t, currentYear]);

  useEffect(() => {
    loadFooter();
  }, []);

  return (
    <div className="w-full mt-3" style={{ border: 'none', outline: 'none', boxShadow: 'none' }}>
      {footer ? (
        <div
          className="custom-footer"
          style={{ border: 'none', outline: 'none', boxShadow: 'none' }}
          dangerouslySetInnerHTML={{ __html: footer }}
        ></div>
      ) : (
        customFooter
      )}
    </div>
  );
};

export default FooterBar;
