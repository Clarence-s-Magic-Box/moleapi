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

import React from 'react';
import { Link } from 'react-router-dom';
import { IconFile, IconInfoCircle } from '@douyinfe/semi-icons';
import { IconIntro, IconModal, IconTag } from '@douyinfe/semi-icons-lab';
import SkeletonWrapper from '../components/SkeletonWrapper';

const Navigation = ({
  mainNavLinks,
  isMobile,
  isLoading,
  userState,
  pricingRequireAuth,
}) => {
  const getNavIcon = (itemKey) => {
    switch (itemKey) {
      case 'home':
        return (
          <IconIntro className='text-blue-500 transition-colors duration-300 hover:text-blue-600' />
        );
      case 'console':
        return (
          <IconModal className='text-sky-500 transition-colors duration-300 hover:text-sky-600' />
        );
      case 'pricing':
        return (
          <IconTag className='text-red-500 transition-colors duration-300 hover:text-red-600' />
        );
      case 'docs':
        return (
          <IconFile className='text-amber-500 transition-colors duration-300 hover:text-amber-600' />
        );
      case 'about':
        return (
          <IconInfoCircle className='text-emerald-500 transition-colors duration-300 hover:text-emerald-600' />
        );
      default:
        return null;
    }
  };

  const renderNavLinks = () => {
    const commonLinkClasses = 'nav-link-item';

    return mainNavLinks.map((link) => {
      const icon = getNavIcon(link.itemKey);
      const linkContent = (
        <>
          {icon && <span className='nav-link-icon'>{icon}</span>}
          <span className='nav-link-text'>{link.text}</span>
        </>
      );

      if (link.isExternal) {
        return (
          <a
            key={link.itemKey}
            href={link.externalLink}
            target='_blank'
            rel='noopener noreferrer'
            className={commonLinkClasses}
          >
            {linkContent}
          </a>
        );
      }

      let targetPath = link.to;
      if (link.itemKey === 'console' && !userState.user) {
        targetPath = '/login';
      }
      if (link.itemKey === 'pricing' && pricingRequireAuth && !userState.user) {
        targetPath = '/login';
      }

      return (
        <Link key={link.itemKey} to={targetPath} className={commonLinkClasses}>
          {linkContent}
        </Link>
      );
    });
  };

  return (
    <nav className='flex flex-1 items-center gap-1 lg:gap-2 mx-2 md:mx-4 overflow-x-auto whitespace-nowrap scrollbar-hide'>
      <SkeletonWrapper
        loading={isLoading}
        type='navigation'
        count={4}
        width={60}
        height={16}
        isMobile={isMobile}
      >
        {renderNavLinks()}
      </SkeletonWrapper>
    </nav>
  );
};

export default Navigation;
