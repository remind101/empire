case "$1" in 
  "apis")
    list_apis
    exit 0
    ;;
  "api-add")
    if [ -z $2 ]; then
      printf "%s\n" "Usage: emp api-add target1=url target2=url"
      exit 1
    else
      shift
      api_add "$@"
      exit 0
    fi
    ;;
  "api-set")
    if [ -z $2 ]; then
      printf "%s\n" "Usage: emp api-set target"
      exit 1
    else
      shift
      api_set "$@"
      exit 0
    fi
  ;;
esac

LOCAL_EMPIRE_URL=${config[${config[current]}]}
EMPIRE_URL=${EMPIRE_URL:-$LOCAL_EMPIRE_URL}
HEROKU_API_URL=${EMPIRE_URL} hk "$@"
