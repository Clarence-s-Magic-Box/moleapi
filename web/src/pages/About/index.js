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
    padding: '24px'
  };

  const customDescription = (
    <div style={{ textAlign: 'center', fontSize: '14px' }}>
      <p>© 2025 MoleAPI. All rights reserved.</p>
      <p>
        基于 <a
          href='https://github.com/Calcium-Ion/new-api'
          target="_blank"
          rel="noopener noreferrer"
          className="!text-semi-color-primary"
        >
          New-API
        </a> © 2025 CalciumIon | 源自 <a
          href='https://github.com/songquanpeng/one-api'
          target="_blank"
          rel="noopener noreferrer"
          className="!text-semi-color-primary"
        >
          One API
        </a> v0.5.4 © 2023 JustSong
      </p>
      <p>
        本项目根据 <a
          href='https://opensource.org/license/mit'
          target="_blank"
          rel="noopener noreferrer"
          className="!text-semi-color-primary"
        >
          MIT许可证
        </a> 授权，在遵守 <a
          href='https://github.com/Calcium-Ion/new-api/blob/main/LICENSE'
          target="_blank"
          rel="noopener noreferrer"
          className="!text-semi-color-primary"
        >
          Apache-2.0协议
        </a> 的前提下使用。
      </p>
      <p>
        根据<a
          href='https://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm'
          target="_blank"
          rel="noopener noreferrer"
          className="!text-semi-color-primary"
        >
          《生成式人工智能服务管理暂行办法》
        </a>的要求，本站不向中国大陆公众提供未经备案的生成式人工智能服务。
      </p>
      <p>
        本站不提供任何形式的人工智能服务，仅提供 API 转发接口，请勿用于任何违法违规用途，否则将承担相应的法律责任。
      </p>
      <p>
        使用本站任何服务时，请务必遵守相应大模型提供商的条款和协议，并在使用时遵守当地法律法规。
      </p>
    </div>
  );

  return (
    <div className="mt-[60px] px-2">
      {aboutLoaded && about === '' ? (
        <div className="flex justify-center items-center h-screen p-8">
          <Empty
            image={<IllustrationConstruction style={{ width: 150, height: 150 }} />}
            darkModeImage={<IllustrationConstructionDark style={{ width: 150, height: 150 }} />}
            description={t('关于本站')}
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
              style={{ fontSize: '16px' }}
              dangerouslySetInnerHTML={{ __html: about }}
            ></div>
          )}
        </>
      )}
    </div>
  );
};

export default About;
