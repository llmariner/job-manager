for line in open('server.go'):
    n= line.find('func (s *S)')
    if n >= 0:
        st = 12
        last = line.find('w http.ResponseWriter')
        func = line[st:last-1]
        print('// %s implements the endpoint' % func)
    print(line.strip())
